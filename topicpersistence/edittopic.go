package topicpersistence

import (
	"os"
	"path"
	"slices"
)

func UpdateTopicFile(topic TopicName, questionsToAdd map[string]string) error {
	questionsDir := QuestionsDir()

	filepath := path.Join(questionsDir, string(topic))
	file, err := os.Create(filepath)
	if err != nil {
		panic(err)
	}

	appendQuestionToFile(file, questionsToAdd)
	err = file.Close()
	if err != nil {
		panic(err)
	}
	return err
}

func UpdateTopicNames(topics []TopicName) bool {
	origTopicNames := RetrieveTopicNames()
	topicsToDelete := []TopicName{}
	for _, topic := range origTopicNames {
		if slices.Index(topics, topic) < 0 {
			topicsToDelete = append(topicsToDelete, topic)
		}
	}

	if len(topicsToDelete) > 1 || len(topicsToDelete) == 0 {
		return false
	}

	topicsToAdd := []TopicName{}
	for _, topic := range topics {
		if slices.Index(origTopicNames, topic) < 0 {
			topicsToAdd = append(topicsToAdd, topic)
		}
	}

	if len(topicsToAdd) > 1 || len(topicsToAdd) == 0 {
		return false
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	topicsDir := path.Join(homeDir, PathToQuestionsDir)
	err = os.Rename(
		path.Join(topicsDir, string(topicsToDelete[0])),
		path.Join(topicsDir, string(topicsToAdd[0])))
	return err == nil
}
