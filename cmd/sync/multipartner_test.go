package main

import (
	"net/http"
	"os"
	"path"
	persistence "primotibalt/checkTests/topicpersistence"
	"slices"
	"sync"
	"testing"
	"time"
)

var newest = time.Date(2026, 12, 1, 12, 0, 0, 0, time.UTC)

func TestPartnerURLs(t *testing.T) {
	cases := []struct {
		raw  string
		want []string
	}{
		{"100.64.0.1", []string{"http://100.64.0.1:8081"}},
		{"100.64.0.1,100.64.0.2", []string{"http://100.64.0.1:8081", "http://100.64.0.2:8081"}},
		{" 100.64.0.1 , 100.64.0.2 ", []string{"http://100.64.0.1:8081", "http://100.64.0.2:8081"}},
		{"100.64.0.1,,100.64.0.2,", []string{"http://100.64.0.1:8081", "http://100.64.0.2:8081"}},
		{"100.64.0.1,100.64.0.1", []string{"http://100.64.0.1:8081"}},
	}

	for _, testcase := range cases {
		t.Setenv("TAILSCALE_PARTNER_IP", testcase.raw)
		got, err := PartnerURLs()
		if err != nil {
			t.Errorf("PartnerURLs(%q) failed: %v", testcase.raw, err)
			continue
		}
		if !slices.Equal(got, testcase.want) {
			t.Errorf("PartnerURLs(%q) = %v, want %v", testcase.raw, got, testcase.want)
		}
	}

	for _, raw := range []string{"", "  ", ",", " , "} {
		t.Setenv("TAILSCALE_PARTNER_IP", raw)
		if _, err := PartnerURLs(); err == nil {
			t.Errorf("PartnerURLs(%q) should have failed", raw)
		}
	}
}

// Every partner contributes what it alone has, and the newest copy of a topic
// several partners disagree about is the one that survives locally.
func TestReconcileWithSeveralPartners(t *testing.T) {
	useTempQuestionsDir(t)
	writeLocalTopic(t, "OnlyLocal", "mine\n", older)
	writeLocalTopic(t, "Shared", "oldest\n", older)

	first, firstServer := newFakePartner(t, map[string]TopicPayload{
		"FromFirst": {Name: "FromFirst", Content: "first\n", Modified: newer},
		"Shared":    {Name: "Shared", Content: "middle\n", Modified: newer},
	})
	second, secondServer := newFakePartner(t, map[string]TopicPayload{
		"FromSecond": {Name: "FromSecond", Content: "second\n", Modified: newer},
		"Shared":     {Name: "Shared", Content: "winner\n", Modified: newest},
	})

	reconcileAll(&http.Client{Timeout: time.Second}, []string{firstServer.URL, secondServer.URL})

	if got := readLocalTopic(t, "FromFirst"); got != "first\n" {
		t.Errorf("FromFirst not pulled, got %q", got)
	}
	if got := readLocalTopic(t, "FromSecond"); got != "second\n" {
		t.Errorf("FromSecond not pulled, got %q", got)
	}

	// Both partners offered Shared at once; the newer one has to win whichever
	// of the two goroutines got there first.
	if got := readLocalTopic(t, "Shared"); got != "winner\n" {
		t.Errorf("Shared = %q, want the newest copy %q", got, "winner\n")
	}
	info, err := os.Stat(path.Join(persistence.QuestionsDir(), "Shared"))
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(newest) {
		t.Errorf("Shared kept mtime %v, want %v", info.ModTime(), newest)
	}

	// A topic only this machine has goes out to every partner.
	for name, partner := range map[string]*fakePartner{"first": first, "second": second} {
		if _, pushed := partner.wasPushed("OnlyLocal"); !pushed {
			t.Errorf("OnlyLocal was not pushed to the %s partner", name)
		}
	}
}

// One dead partner must not stop the others from syncing.
func TestReconcileSkipsOfflinePartner(t *testing.T) {
	useTempQuestionsDir(t)

	_, deadServer := newFakePartner(t, map[string]TopicPayload{})
	deadURL := deadServer.URL
	deadServer.Close()

	_, liveServer := newFakePartner(t, map[string]TopicPayload{
		"Alive": {Name: "Alive", Content: "alive\n", Modified: newer},
	})

	reconcileAll(&http.Client{Timeout: time.Second}, []string{deadURL, liveServer.URL})

	if got := readLocalTopic(t, "Alive"); got != "alive\n" {
		t.Errorf("live partner was not synced, got %q", got)
	}
}

func TestWriteTopicKeepsTheNewerCopy(t *testing.T) {
	useTempQuestionsDir(t)
	writeLocalTopic(t, "Topic", "current\n", newer)

	// An older update from a lagging partner must not win.
	applied, err := WriteTopic(TopicPayload{Name: "Topic", Content: "stale\n", Modified: older})
	if err != nil {
		t.Fatal(err)
	}
	if applied {
		t.Error("an older payload overwrote the newer local copy")
	}
	if got := readLocalTopic(t, "Topic"); got != "current\n" {
		t.Errorf("local copy became %q", got)
	}

	// The same content is never rewritten, whatever its timestamp says.
	applied, err = WriteTopic(TopicPayload{Name: "Topic", Content: "current\n", Modified: newest})
	if err != nil {
		t.Fatal(err)
	}
	if applied {
		t.Error("identical content was written again")
	}

	// A genuinely newer update does land.
	applied, err = WriteTopic(TopicPayload{Name: "Topic", Content: "newest\n", Modified: newest})
	if err != nil {
		t.Fatal(err)
	}
	if !applied {
		t.Error("a newer payload was rejected")
	}
	if got := readLocalTopic(t, "Topic"); got != "newest\n" {
		t.Errorf("local copy = %q, want %q", got, "newest\n")
	}
}

// Several partners writing the same topic at once must leave a whole file
// holding the newest version — never a mix of two writes. Run with -race.
func TestConcurrentWritesToTheSameTopic(t *testing.T) {
	useTempQuestionsDir(t)

	const partners = 12
	payloads := make([]TopicPayload, partners)
	for i := range payloads {
		payloads[i] = TopicPayload{
			Name:     "Topic",
			Content:  "version " + string(rune('a'+i)) + "\n",
			Modified: older.Add(time.Duration(i) * time.Hour),
		}
	}
	winner := payloads[partners-1]

	var waitgroup sync.WaitGroup
	for _, payload := range payloads {
		waitgroup.Add(1)
		go func() {
			defer waitgroup.Done()
			if _, err := WriteTopic(payload); err != nil {
				t.Error(err)
			}
		}()
		waitgroup.Add(1)
		go func() {
			defer waitgroup.Done()
			// Readers race the writers: each one must see a complete file.
			read, err := ReadTopic("Topic")
			if err != nil {
				return // not created yet, which is fine
			}
			if read.Content != "" && !slices.ContainsFunc(payloads, func(p TopicPayload) bool {
				return p.Content == read.Content
			}) {
				t.Errorf("read a torn file: %q", read.Content)
			}
		}()
	}
	waitgroup.Wait()

	if got := readLocalTopic(t, "Topic"); got != winner.Content {
		t.Errorf("final content %q, want the newest %q", got, winner.Content)
	}
	info, err := os.Stat(path.Join(persistence.QuestionsDir(), "Topic"))
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(winner.Modified) {
		t.Errorf("final mtime %v, want %v", info.ModTime(), winner.Modified)
	}

	// The temporary files used for the atomic writes are all cleaned up.
	entries, err := os.ReadDir(persistence.QuestionsDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if isHiddenName(entry.Name()) {
			t.Errorf("temporary file %s was left behind", entry.Name())
		}
	}
}

func TestFilename(t *testing.T) {
	cases := map[string]string{
		"/home/user/.local/share/primotibalt/Questions/Topic": "Topic",
		"Topic":                               "Topic",
		"/home/user/Questions/.Topic.sync123": ".Topic.sync123",
	}
	for eventName, want := range cases {
		if got := filename(eventName); got != want {
			t.Errorf("filename(%q) = %q, want %q", eventName, got, want)
		}
	}
}
