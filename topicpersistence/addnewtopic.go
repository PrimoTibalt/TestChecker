// Package topicpersistence
package topicpersistence

import (
	"os"
	"path"
)

func AddNewTopic(topic TopicName) (filepath string, err error) {
	questionsDir := QuestionsDir()

	filepath = path.Join(questionsDir, string(topic))
	file, err := os.Create(filepath)
	if err != nil {
		panic(err)
	}
	err = file.Close()
	if err != nil {
		panic(err)
	}
	return
}
