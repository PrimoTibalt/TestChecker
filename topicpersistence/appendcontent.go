package topicpersistence

import (
	"os"
	"path"
)

func AppendQuestionsToTopic(topicName TopicName, questionsToAdd map[string]string) {
	questionsDir := QuestionsDir()

	filepath := path.Join(questionsDir, string(topicName))
	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		panic(err)
	}

	defer file.Close()

	if err = appendQuestionToFile(file, questionsToAdd); err != nil {
		panic(err)
	}
}

func appendQuestionToFile(file *os.File, questionsToAdd map[string]string) error {
	for q, a := range questionsToAdd {
		_, err := file.Write([]byte(FormatQaLine(q, a)))
		if err != nil {
			return err
		}
	}

	return nil
}
