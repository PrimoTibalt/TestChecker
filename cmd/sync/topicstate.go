package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path"
	"path/filepath"
	persistence "primotibalt/checkTests/topicpersistence"
	"strings"
	"time"
)

// TopicState describes one topic file as it exists on a single machine. It is
// what the instances exchange to work out who has the fresher copy.
type TopicState struct {
	Name     string    `json:"name"`
	Modified time.Time `json:"modified"`
	Hash     string    `json:"hash"`
}

// TopicPayload carries the whole content of a topic from one machine to the
// other, together with the modification time that should be preserved on the
// receiving side.
type TopicPayload struct {
	Name     string    `json:"name"`
	Content  string    `json:"content"`
	Modified time.Time `json:"modified"`
}

// LocalTopics lists every topic file in the local Questions dir with its
// modification time and a hash of its content.
func LocalTopics() ([]TopicState, error) {
	questionsDir := persistence.QuestionsDir()
	if err := os.MkdirAll(questionsDir, 0o755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(questionsDir)
	if err != nil {
		return nil, err
	}

	topics := make([]TopicState, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || isHiddenName(entry.Name()) {
			continue
		}
		state, ok, err := LocalTopicState(entry.Name())
		if err != nil {
			return nil, err
		}
		if ok {
			topics = append(topics, state)
		}
	}

	return topics, nil
}

// LocalTopicState reads the current state of a single topic. Sync decisions are
// taken against a listing that may already be stale by the time we act on it,
// so anything about to copy a file checks the state again through here.
func LocalTopicState(name string) (TopicState, bool, error) {
	state := TopicState{}
	filepath, err := TopicPath(name)
	if err != nil {
		return state, false, err
	}

	unlock := topicLocks.Lock(name)
	defer unlock()

	info, err := os.Stat(filepath)
	if errors.Is(err, os.ErrNotExist) {
		return state, false, nil
	}
	if err != nil {
		return state, false, err
	}
	content, err := os.ReadFile(filepath)
	if err != nil {
		return state, false, err
	}

	state.Name = name
	state.Modified = info.ModTime()
	state.Hash = hashContent(content)
	return state, true, nil
}

// ReadTopic loads a single topic file into a payload ready to be sent over.
func ReadTopic(name string) (TopicPayload, error) {
	payload := TopicPayload{}
	filepath, err := TopicPath(name)
	if err != nil {
		return payload, err
	}

	unlock := topicLocks.Lock(name)
	defer unlock()

	content, err := os.ReadFile(filepath)
	if err != nil {
		return payload, err
	}
	info, err := os.Stat(filepath)
	if err != nil {
		return payload, err
	}

	payload.Name = name
	payload.Content = string(content)
	payload.Modified = info.ModTime()
	return payload, nil
}

// WriteTopic stores a payload received from a partner and reports whether it
// actually replaced anything. It refuses to overwrite a copy that is already
// newer than the one being offered, which is what stops several partners
// pushing the same topic at once from undoing each other: the newest version
// wins no matter what order the requests arrive in. The write itself goes
// through a temporary file so a reader never sees a half-written topic.
func WriteTopic(payload TopicPayload) (bool, error) {
	filepath, err := TopicPath(payload.Name)
	if err != nil {
		return false, err
	}

	unlock := topicLocks.Lock(payload.Name)
	defer unlock()

	info, err := os.Stat(filepath)
	switch {
	case err == nil:
		current, readErr := os.ReadFile(filepath)
		if readErr != nil {
			return false, readErr
		}
		if hashContent(current) == hashContent([]byte(payload.Content)) {
			return false, nil
		}
		if !payload.Modified.IsZero() && info.ModTime().After(payload.Modified) {
			return false, nil
		}
	case !errors.Is(err, os.ErrNotExist):
		return false, err
	}

	if err = writeFileAtomically(filepath, []byte(payload.Content)); err != nil {
		return false, err
	}
	if payload.Modified.IsZero() {
		return true, nil
	}
	return true, os.Chtimes(filepath, payload.Modified, payload.Modified)
}

// writeFileAtomically replaces a topic in one step, so a concurrent reader gets
// either the old file or the new one and never a truncated mix of the two.
func writeFileAtomically(topicpath string, content []byte) error {
	temp, err := os.CreateTemp(path.Dir(topicpath), "."+path.Base(topicpath)+".sync")
	if err != nil {
		return err
	}
	tempname := temp.Name()
	defer os.Remove(tempname)

	if _, err = temp.Write(content); err != nil {
		temp.Close()
		return err
	}
	if err = temp.Chmod(0o644); err != nil {
		temp.Close()
		return err
	}
	if err = temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempname, topicpath)
}

// RemoveTopicFile deletes a topic under its lock, so it cannot land in the
// middle of another partner's write.
func RemoveTopicFile(name string) error {
	filepath, err := TopicPath(name)
	if err != nil {
		return err
	}

	unlock := topicLocks.Lock(name)
	defer unlock()

	err = os.Remove(filepath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// TopicPath resolves a topic name to a path inside the Questions dir. Names
// arrive over the network, so anything that is not a plain file name is
// rejected instead of being allowed to escape the directory.
func TopicPath(name string) (string, error) {
	if name == "" || name != filepath.Base(name) || name == "." || name == ".." {
		return "", errors.New("invalid topic name " + name)
	}
	return path.Join(persistence.QuestionsDir(), name), nil
}

// isHiddenName tells apart our own half-written temporary files (and editor
// leftovers) from real topics.
func isHiddenName(name string) bool {
	return strings.HasPrefix(name, ".")
}

func hashContent(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func indexTopics(topics []TopicState) map[string]TopicState {
	byName := make(map[string]TopicState, len(topics))
	for _, topic := range topics {
		byName[topic.Name] = topic
	}
	return byName
}
