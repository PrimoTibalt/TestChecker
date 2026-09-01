package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	persistence "primotibalt/checkTests/topicpersistence"
	"sync"
)

func RunServer(breaker chan bool, notifier chan bool) {
	myaddr, err := LocalAddr()
	if err != nil {
		panic(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /createTopic", breakWatcher(createNewTopic, breaker))
	mux.HandleFunc("POST /removeTopic", breakWatcher(removeTopic, breaker))
	mux.HandleFunc("POST /changeTopic", breakWatcher(changeTopic, breaker))
	mux.HandleFunc("GET /topics", listTopics)
	mux.HandleFunc("GET /topic", sendTopic)
	mux.HandleFunc("POST /syncTopic", breakWatcher(receiveTopic, breaker))
	http.ListenAndServe(myaddr, mux)
}

// watcherLock keeps two handlers from stopping and re-attaching the watcher at
// the same time — a startup sync sends topics in a burst, so the requests do
// arrive together.
var watcherLock sync.Mutex

func breakWatcher(handler func(http.ResponseWriter, *http.Request), breaker chan bool) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		watcherLock.Lock()
		defer watcherLock.Unlock()
		breaker <- true
		handler(rw, r)
		AttachWatcher(persistence.QuestionsDir(), breaker)
	}
}

// listTopics answers the partner's startup question: what topics do you have
// and when was each of them last changed.
func listTopics(rw http.ResponseWriter, r *http.Request) {
	topics, err := LocalTopics()
	if err != nil {
		fmt.Println("could not list topics")
		fmt.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	body, err := json.Marshal(topics)
	if err != nil {
		fmt.Println("could not marshal topics")
		fmt.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.Write(body)
}

// sendTopic hands one whole topic over to the partner.
func sendTopic(rw http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	payload, err := ReadTopic(name)
	if err != nil {
		fmt.Println("could not read topic " + name)
		fmt.Println(err)
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	body, err := json.Marshal(&payload)
	if err != nil {
		fmt.Println("could not marshal topic " + name)
		fmt.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.Write(body)
}

// receiveTopic stores a topic the partner considers newer than ours.
func receiveTopic(rw http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println("could not read body")
		fmt.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	payload := TopicPayload{}
	if err = json.Unmarshal(body, &payload); err != nil {
		fmt.Println("could not unmarshal payload")
		fmt.Println(err)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	applied, err := WriteTopic(payload)
	if err != nil {
		fmt.Println("could not save topic " + payload.Name)
		fmt.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	if applied {
		fmt.Println("took " + payload.Name + " from partner")
	} else {
		fmt.Println("kept the local " + payload.Name + ", it is not older than the partner's copy")
	}
	// Either way the partner is up to date as far as this topic goes.
	rw.WriteHeader(http.StatusOK)
}

func changeTopic(rw http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	createTopicModel, err := getcreatetopicmodel(r, rw)
	if err != nil {
		return
	}

	// A live edit from a partner's watcher carries no timestamp, so it is
	// applied as the newest version there is — but still atomically and under
	// the topic's lock, so it cannot collide with a startup sync in flight.
	name := filename(createTopicModel.Name)
	unlock := topicLocks.Lock(name)
	defer unlock()

	topicpath, err := TopicPath(name)
	if err != nil {
		fmt.Println(err)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	if err = writeFileAtomically(topicpath, []byte(createTopicModel.Content)); err != nil {
		fmt.Println("could not write to file " + name)
		fmt.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	rw.WriteHeader(http.StatusOK)
}

func getcreatetopicmodel(r *http.Request, rw http.ResponseWriter) (CreateTopic, error) {
	defer r.Body.Close()
	createTopicModel := CreateTopic{}
	allBytes, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println(err)
		fmt.Println("could not create new topic")
		rw.WriteHeader(http.StatusInternalServerError)
		return createTopicModel, errors.New("Heh")
	}
	err = json.Unmarshal(allBytes, &createTopicModel)
	if err != nil {
		fmt.Println(err)
		fmt.Println("could not unmarchal payload")
		rw.WriteHeader(http.StatusInternalServerError)
		return createTopicModel, errors.New("Heh")
	}
	return createTopicModel, nil
}

func createNewTopic(rw http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	createTopicModel, err := getcreatetopicmodel(r, rw)
	if err != nil {
		return
	}
	name := filename(createTopicModel.Name)
	unlock := topicLocks.Lock(name)
	defer unlock()

	topicpath, err := TopicPath(name)
	if err != nil {
		fmt.Println(err)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	// A partner announcing a brand new topic must not blank out one we already
	// have — that is the empty create event arriving after our own content.
	if _, err = os.Stat(topicpath); err == nil && createTopicModel.Content == "" {
		fmt.Println("already have " + name + ", ignoring the empty create")
		rw.WriteHeader(http.StatusOK)
		return
	}

	if err = writeFileAtomically(topicpath, []byte(createTopicModel.Content)); err != nil {
		fmt.Println("could not create topic " + name)
		fmt.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	rw.WriteHeader(http.StatusOK)
}

func removeTopic(rw http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println("could not read body")
		fmt.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	name := filename(string(body))
	if err = RemoveTopicFile(name); err != nil {
		fmt.Println("could not remove " + name)
		fmt.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	rw.WriteHeader(http.StatusOK)
}
