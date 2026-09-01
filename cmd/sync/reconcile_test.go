package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	persistence "primotibalt/checkTests/topicpersistence"
	"sync"
	"testing"
	"time"
)

var (
	older = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	newer = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
)

// fakePartner serves a fixed set of topics and records everything pushed to it.
type fakePartner struct {
	mu       sync.Mutex
	topics   map[string]TopicPayload
	received map[string]TopicPayload
}

func newFakePartner(t *testing.T, topics map[string]TopicPayload) (*fakePartner, *httptest.Server) {
	t.Helper()
	partner := &fakePartner{topics: topics, received: map[string]TopicPayload{}}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /topics", func(rw http.ResponseWriter, r *http.Request) {
		partner.mu.Lock()
		defer partner.mu.Unlock()
		states := []TopicState{}
		for name, payload := range partner.topics {
			states = append(states, TopicState{
				Name:     name,
				Modified: payload.Modified,
				Hash:     hashContent([]byte(payload.Content)),
			})
		}
		json.NewEncoder(rw).Encode(states)
	})
	mux.HandleFunc("GET /topic", func(rw http.ResponseWriter, r *http.Request) {
		partner.mu.Lock()
		defer partner.mu.Unlock()
		payload, ok := partner.topics[r.URL.Query().Get("name")]
		if !ok {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		json.NewEncoder(rw).Encode(&payload)
	})
	mux.HandleFunc("POST /syncTopic", func(rw http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := io.ReadAll(r.Body)
		payload := TopicPayload{}
		if err := json.Unmarshal(body, &payload); err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		partner.mu.Lock()
		defer partner.mu.Unlock()
		partner.received[payload.Name] = payload
		rw.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return partner, server
}

func (p *fakePartner) wasPushed(name string) (TopicPayload, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	payload, ok := p.received[name]
	return payload, ok
}

func (p *fakePartner) forgetPushes() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.received = map[string]TopicPayload{}
}

// useTempQuestionsDir points topicpersistence at a throwaway home directory so
// the tests never touch the real Questions dir.
func useTempQuestionsDir(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	questionsDir := persistence.QuestionsDir()
	if err := os.MkdirAll(questionsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return questionsDir
}

func writeLocalTopic(t *testing.T, name string, content string, modified time.Time) {
	t.Helper()
	filepath := path.Join(persistence.QuestionsDir(), name)
	if err := os.WriteFile(filepath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath, modified, modified); err != nil {
		t.Fatal(err)
	}
}

func readLocalTopic(t *testing.T, name string) string {
	t.Helper()
	content, err := os.ReadFile(path.Join(persistence.QuestionsDir(), name))
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func TestReconcile(t *testing.T) {
	useTempQuestionsDir(t)

	writeLocalTopic(t, "OnlyLocal", "q1"+persistence.DelimeterQuestionAnswer+"a1\n", older)
	writeLocalTopic(t, "LocalIsNewer", "fresh\n", newer)
	writeLocalTopic(t, "PartnerIsNewer", "stale\n", older)
	writeLocalTopic(t, "Same", "identical\n", older)
	writeLocalTopic(t, "SameContentOtherTime", "identical\n", newer)

	partner, server := newFakePartner(t, map[string]TopicPayload{
		"OnlyPartner":          {Name: "OnlyPartner", Content: "from partner\n", Modified: older},
		"LocalIsNewer":         {Name: "LocalIsNewer", Content: "stale\n", Modified: older},
		"PartnerIsNewer":       {Name: "PartnerIsNewer", Content: "fresh\n", Modified: newer},
		"Same":                 {Name: "Same", Content: "identical\n", Modified: older},
		"SameContentOtherTime": {Name: "SameContentOtherTime", Content: "identical\n", Modified: older},
	})

	reconcileWith(server.Client(), server.URL)

	// Pulled from the partner.
	if got := readLocalTopic(t, "OnlyPartner"); got != "from partner\n" {
		t.Errorf("OnlyPartner not pulled, got %q", got)
	}
	if got := readLocalTopic(t, "PartnerIsNewer"); got != "fresh\n" {
		t.Errorf("PartnerIsNewer not pulled, got %q", got)
	}

	// Pushed to the partner.
	if pushed, _ := partner.wasPushed("OnlyLocal"); pushed.Content != "q1"+persistence.DelimeterQuestionAnswer+"a1\n" {
		t.Errorf("OnlyLocal not pushed, got %q", pushed.Content)
	}
	if pushed, _ := partner.wasPushed("LocalIsNewer"); pushed.Content != "fresh\n" {
		t.Errorf("LocalIsNewer not pushed, got %q", pushed.Content)
	}

	// Untouched: same content on both sides.
	for _, name := range []string{"Same", "SameContentOtherTime"} {
		if _, pushed := partner.wasPushed(name); pushed {
			t.Errorf("%s was pushed even though the content matches", name)
		}
	}

	// Nothing that only lives on the local side may be deleted.
	if got := readLocalTopic(t, "OnlyLocal"); got == "" {
		t.Error("OnlyLocal was emptied")
	}
}

// A pulled topic keeps the partner's modification time, so a second run has
// nothing left to do.
func TestReconcileIsIdempotent(t *testing.T) {
	useTempQuestionsDir(t)
	writeLocalTopic(t, "Topic", "stale\n", older)

	partner, server := newFakePartner(t, map[string]TopicPayload{
		"Topic": {Name: "Topic", Content: "fresh\n", Modified: newer},
	})

	reconcileWith(server.Client(), server.URL)
	if got := readLocalTopic(t, "Topic"); got != "fresh\n" {
		t.Fatalf("first run did not pull, got %q", got)
	}

	partner.forgetPushes()
	reconcileWith(server.Client(), server.URL)
	if _, pushed := partner.wasPushed("Topic"); pushed {
		t.Error("second run pushed the topic back at the partner")
	}
}

func TestReconcileWithOfflinePartner(t *testing.T) {
	useTempQuestionsDir(t)
	writeLocalTopic(t, "Topic", "mine\n", older)

	_, server := newFakePartner(t, map[string]TopicPayload{})
	offlineURL := server.URL
	server.Close()

	reconcileWith(&http.Client{Timeout: time.Second}, offlineURL)

	if got := readLocalTopic(t, "Topic"); got != "mine\n" {
		t.Errorf("local topic changed while partner was offline, got %q", got)
	}
}

func TestTopicPathRejectsEscapes(t *testing.T) {
	useTempQuestionsDir(t)

	for _, name := range []string{"", ".", "..", "../escape", "sub/topic", "/etc/passwd"} {
		if _, err := TopicPath(name); err == nil {
			t.Errorf("TopicPath accepted %q", name)
		}
	}

	if _, err := TopicPath("Topic"); err != nil {
		t.Errorf("TopicPath rejected a plain name: %v", err)
	}
}

// The handlers the partner talks to are exercised against a real local dir.
func TestServerHandlers(t *testing.T) {
	useTempQuestionsDir(t)
	writeLocalTopic(t, "Topic", "content\n", older)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /topics", listTopics)
	mux.HandleFunc("GET /topic", sendTopic)
	mux.HandleFunc("POST /syncTopic", receiveTopic)
	server := httptest.NewServer(mux)
	defer server.Close()

	topics, err := askPartnerForTopics(server.Client(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(topics) != 1 || topics[0].Name != "Topic" {
		t.Fatalf("unexpected listing %v", topics)
	}
	if !topics[0].Modified.Equal(older) {
		t.Errorf("modification time %v, want %v", topics[0].Modified, older)
	}
	if topics[0].Hash != hashContent([]byte("content\n")) {
		t.Error("hash does not match the content")
	}

	response, err := server.Client().Get(server.URL + "/topic?name=Topic")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	payload := TopicPayload{}
	if err = json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Content != "content\n" {
		t.Errorf("got %q from /topic", payload.Content)
	}

	missing, err := server.Client().Get(server.URL + "/topic?name=Missing")
	if err != nil {
		t.Fatal(err)
	}
	defer missing.Body.Close()
	if missing.StatusCode != http.StatusNotFound {
		t.Errorf("/topic answered %s for a missing topic", missing.Status)
	}

	pushTopicPayload(t, server, TopicPayload{Name: "Topic", Content: "replaced\n", Modified: newer})
	if got := readLocalTopic(t, "Topic"); got != "replaced\n" {
		t.Errorf("/syncTopic did not overwrite, got %q", got)
	}
	info, err := os.Stat(path.Join(persistence.QuestionsDir(), "Topic"))
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(newer) {
		t.Errorf("/syncTopic did not keep the partner's time, got %v", info.ModTime())
	}

	pushTopicPayload(t, server, TopicPayload{Name: "Created", Content: "new\n", Modified: newer})
	if got := readLocalTopic(t, "Created"); got != "new\n" {
		t.Errorf("/syncTopic did not create the topic, got %q", got)
	}
}

func pushTopicPayload(t *testing.T, server *httptest.Server, payload TopicPayload) {
	t.Helper()
	body, err := json.Marshal(&payload)
	if err != nil {
		t.Fatal(err)
	}
	response, err := server.Client().Post(server.URL+"/syncTopic", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("/syncTopic answered %s", response.Status)
	}
}
