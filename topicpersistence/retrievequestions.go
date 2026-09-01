// Package topicpersistence
package topicpersistence

import (
	"os"
	"path"
	"strings"
)

const (
	DelimeterQuestionAnswer = "/!/"
	// DelimeterNewLine stands in for a line break inside a question or an
	// answer, so that a code snippet still occupies exactly one line on disk.
	DelimeterNewLine     = "/!n/"
	PathToQuestionsDir   = "/.local/share/primotibalt/Questions"
	directoryPermissions = 0o755
)

type TopicName string

// FormatQaLine renders a pair as the single on-disk line `question/!/answer`.
func FormatQaLine(question, answer string) string {
	return encodeQaField(question) + DelimeterQuestionAnswer + encodeQaField(answer) + "\n"
}

// ParseQaLine splits an on-disk line back into a pair, restoring line breaks.
// It reports false for lines that hold no pair at all, such as the trailing
// empty line every topic file ends with.
func ParseQaLine(line string) (question, answer string, ok bool) {
	qa := strings.SplitN(line, DelimeterQuestionAnswer, 2)
	if len(qa) < 2 {
		return "", "", false
	}

	return decodeQaField(qa[0]), decodeQaField(qa[1]), true
}

func encodeQaField(field string) string {
	return strings.ReplaceAll(strings.ReplaceAll(field, "\r\n", "\n"), "\n", DelimeterNewLine)
}

func decodeQaField(field string) string {
	return strings.ReplaceAll(field, DelimeterNewLine, "\n")
}

func RetrieveTopicNames() (topics []TopicName) {
	pathToTopics := QuestionsDir()
	if _, err := os.Stat(pathToTopics); err != nil {
		if err = os.MkdirAll(pathToTopics, directoryPermissions); err != nil {
			panic(err)
		}
	}

	questionsDirEntries, getqInfoErr := os.ReadDir(pathToTopics)
	if getqInfoErr != nil {
		panic(getqInfoErr)
	}

	topics = make([]TopicName, len(questionsDirEntries))
	for ptr, dir := range questionsDirEntries {
		topics[ptr] = TopicName(dir.Name())
	}

	return
}

func TopicQuestions(topic TopicName) (qaPairs []string) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	pathToTopic := path.Join(homeDir, PathToQuestionsDir, string(topic))
	fi, fiErr := os.ReadFile(pathToTopic)
	if fiErr != nil {
		panic(fiErr)
	}

	content := string(fi)
	return strings.Split(content, "\n")
}

func QuestionsDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return path.Join(homeDir, PathToQuestionsDir)
}
