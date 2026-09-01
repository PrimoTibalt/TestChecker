// Package topicpersistence
package topicpersistence

import (
	"os"
	"path"
)

func RemoveTopic(topicName TopicName) (ok bool, err error) {
	questionsDir := QuestionsDir()

	filepath := path.Join(questionsDir, string(topicName))
	err = os.Remove(filepath)
	if err != nil {
		return false, err
	}

	return true, nil
}
