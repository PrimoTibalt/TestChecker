package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const reconcileTimeout = 10 * time.Second

// Reconcile brings this instance and every partner to the same set of topics on
// startup. Each partner is handled by its own goroutine, so one unreachable
// machine never holds up the others; a partner that does not answer is simply
// skipped and picked up on the next start.
//
// Topics that exist only on one side are copied to the other one, and topics
// that exist on both are resolved by modification time. Deletions are not
// propagated here: without a record of what was deleted, a topic missing on one
// side is indistinguishable from a topic newly added on the other, so removals
// only travel live through the watcher.
func Reconcile() {
	partnerurls, err := PartnerURLs()
	if err != nil {
		fmt.Println("skipping startup sync:", err)
		return
	}

	reconcileAll(&http.Client{Timeout: reconcileTimeout}, partnerurls)
}

func reconcileAll(client *http.Client, partnerurls []string) {
	var waitgroup sync.WaitGroup
	for _, partnerurl := range partnerurls {
		waitgroup.Add(1)
		go func() {
			defer waitgroup.Done()
			reconcileWith(client, partnerurl)
		}()
	}
	waitgroup.Wait()

	fmt.Printf("startup sync finished for all %d partners\n", len(partnerurls))
}

func reconcileWith(client *http.Client, partnerurl string) {
	partnerTopics, err := askPartnerForTopics(client, partnerurl)
	if err != nil {
		report(partnerurl, "partner is not available, skipping startup sync: %v", err)
		return
	}

	localTopics, err := LocalTopics()
	if err != nil {
		report(partnerurl, "could not read local topics: %v", err)
		return
	}

	report(partnerurl, "startup sync: %d local topics, %d partner topics", len(localTopics), len(partnerTopics))

	onlyLocal := indexTopics(localTopics)
	for _, partnerTopic := range partnerTopics {
		delete(onlyLocal, partnerTopic.Name)

		// Another partner's goroutine may have replaced this topic since the
		// listing above, so the comparison uses the state on disk right now.
		localTopic, existsLocally, err := LocalTopicState(partnerTopic.Name)
		if err != nil {
			report(partnerurl, "could not read local %s: %v", partnerTopic.Name, err)
			continue
		}

		switch {
		case !existsLocally:
			report(partnerurl, "new topic on partner: %s", partnerTopic.Name)
			pullTopic(client, partnerurl, partnerTopic.Name)
		case localTopic.Hash == partnerTopic.Hash:
			continue
		case partnerTopic.Modified.After(localTopic.Modified):
			report(partnerurl, "partner has a newer %s", partnerTopic.Name)
			pullTopic(client, partnerurl, partnerTopic.Name)
		case localTopic.Modified.After(partnerTopic.Modified):
			report(partnerurl, "partner has an older %s", partnerTopic.Name)
			pushTopic(client, partnerurl, partnerTopic.Name)
		default:
			report(partnerurl, "conflict: %s differs on both machines but was changed"+
				" at the same time, keeping the local copy", partnerTopic.Name)
		}
	}

	for name := range onlyLocal {
		report(partnerurl, "partner is missing %s", name)
		pushTopic(client, partnerurl, name)
	}

	report(partnerurl, "startup sync finished")
}

// report labels every line with the partner it is about, because the partners
// are synced in parallel and their output is interleaved.
func report(partnerurl string, format string, args ...any) {
	fmt.Printf("["+partnerurl+"] "+format+"\n", args...)
}

func askPartnerForTopics(client *http.Client, partnerurl string) ([]TopicState, error) {
	response, err := client.Get(partnerurl + "/topics")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.New("partner answered with " + response.Status)
	}

	topics := []TopicState{}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(body, &topics); err != nil {
		return nil, err
	}
	return topics, nil
}

func pullTopic(client *http.Client, partnerurl string, name string) {
	response, err := client.Get(partnerurl + "/topic?name=" + url.QueryEscape(name))
	if err != nil {
		report(partnerurl, "could not ask partner for %s: %v", name, err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		report(partnerurl, "partner answered with %s for %s", response.Status, name)
		return
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		report(partnerurl, "could not read partner answer for %s: %v", name, err)
		return
	}

	payload := TopicPayload{}
	if err = json.Unmarshal(body, &payload); err != nil {
		report(partnerurl, "could not unmarshal partner answer for %s: %v", name, err)
		return
	}

	applied, err := WriteTopic(payload)
	if err != nil {
		report(partnerurl, "could not save %s: %v", name, err)
		return
	}
	if !applied {
		report(partnerurl, "kept the local %s, it is not older than the partner's copy", name)
		return
	}
	report(partnerurl, "took %s from partner", name)
}

func pushTopic(client *http.Client, partnerurl string, name string) {
	// Read under the topic lock, so what goes out is a whole file and the
	// newest one we have — including anything another partner just handed us.
	payload, err := ReadTopic(name)
	if err != nil {
		report(partnerurl, "could not read %s: %v", name, err)
		return
	}

	body, err := json.Marshal(&payload)
	if err != nil {
		report(partnerurl, "could not marshal %s: %v", name, err)
		return
	}

	response, err := client.Post(
		partnerurl+"/syncTopic",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		report(partnerurl, "could not send %s to partner: %v", name, err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		report(partnerurl, "partner answered with %s for %s", response.Status, name)
		return
	}
	report(partnerurl, "sent %s to partner", name)
}
