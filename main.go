package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	persistence "primotibalt/checkTests/topicpersistence"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
)

const (
	paddingToLeft           = 4
	defaultTaPlaceholder    = "Напиши ответ на вопрос"
	addNewQuestionToATopic  = "Добавить новый вопрос в топик"
	testKnowledgeOnTheTopic = "Проверить знания по топику"
	addNewTopic             = "Добавить новый топик"
	removeTopic             = "Удалить топик"
	editTopic               = "Редактировать топик"
	Padding                 = 1 // for some reason it just works and prevent first line of the select from disappearing
	fallbackWidth           = 80
	fallbackHeight          = 24
)

type mainOption struct {
	name   string
	action func([]persistence.TopicName)
}

var (
	mainOptions []mainOption
	border                     = lipgloss.BlockBorder()
	borderStyle lipgloss.Style = lipgloss.NewStyle().Border(border, true)
)

type QaPair struct {
	Question string
	Answer   string
}

func init() {
	mainOptions = []mainOption{
		{addNewQuestionToATopic, addNewQuestionToTopicFunc},
		{testKnowledgeOnTheTopic, testKnowledgeOnTheTopicFunc},
		{addNewTopic, addNewTopicFunc},
		{removeTopic, removeTopicFunc},
		{editTopic, editTopicFunc},
	}
}

func main() {
	requestExtendedKeys()
	defer releaseExtendedKeys()

	for {
		var action int
		options := make([]huh.Option[int], len(mainOptions))
		for ptr, option := range mainOptions {
			options[ptr] = huh.NewOption(option.name, ptr)
		}
		err := huh.NewForm(huh.NewGroup(
			huh.NewSelect[int]().
				Options(options...).
				Title("Знания - сила. Что делать будем?").
				Value(&action))).
			WithProgramOptions(tea.WithAltScreen()).Run()
		if err != nil {
			if err != huh.ErrUserAborted {
				panic(err)
			}

			// Returning rather than exiting so the terminal gets its
			// extended-keys setting back.
			return
		}

		topics := persistence.RetrieveTopicNames()
		mainOptions[action].action(topics)
	}
}

func testKnowledgeOnTheTopicFunc(topics []persistence.TopicName) {
	topicToQa := getAllQuestionsForAllTopics(topics)
	selectedTopics := ChooseTopicsForTest(topics, topicToQa)
	if len(selectedTopics) == 0 {
		fmt.Println("Вы не выбрали ничего.")
		return
	}
	RunKnowledgeTest(selectedTopics)
}

func addNewQuestionToTopicFunc(topics []persistence.TopicName) {
	topicToQa := getAllQuestionsForAllTopics(topics)
	selectedTopic, ok := ChooseTopic(topics, topicToQa)
	if !ok {
		return
	}

	RunTopicQuestionAppend(selectedTopic)
}

func addNewTopicFunc(topics []persistence.TopicName) {
	AddNewTopic(topics)
}

func removeTopicFunc(topics []persistence.TopicName) {
	topicToQa := getAllQuestionsForAllTopics(topics)
	RemoveTopic(topics, topicToQa)
}

func editTopicFunc(topics []persistence.TopicName) {
	topicToQA := make(map[persistence.TopicName][]QaPair)
	for _, topic := range topics {
		pairs := []QaPair{}
		for _, qaLine := range persistence.TopicQuestions(topic) {
			question, answer, ok := persistence.ParseQaLine(qaLine)
			if !ok {
				continue
			}
			pairs = append(pairs, QaPair{question, answer})
		}

		topicToQA[topic] = pairs
	}

	EditTopic(topicToQA)
}

func getAllQuestionsForAllTopics(topics []persistence.TopicName) (topicToQa map[persistence.TopicName][]string) {
	topicToQa = map[persistence.TopicName][]string{}
	for _, topic := range topics {
		questions := []string{}
		for _, qaLine := range persistence.TopicQuestions(topic) {
			question, _, ok := persistence.ParseQaLine(qaLine)
			if !ok {
				continue
			}
			questions = append(questions, question)
		}

		topicToQa[topic] = questions
	}

	return
}

// terminalSize falls back to a sane default when the size cannot be read, as
// happens when the output is not a terminal at all.
func terminalSize() (width, height int) {
	width, height, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		return fallbackWidth, fallbackHeight
	}

	return
}

func MoveCursorToTopLeft() {
	command := exec.Command("clear")
	command.Stdout = os.Stdout
	command.Run()
	fmt.Print("\033[H")
}

func MakeThemeCenterInnerElement(theme *huh.Theme) *huh.Theme {
	w, _, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		panic(err)
	}
	theme.Focused.Base = lipgloss.NewStyle().
		AlignHorizontal(lipgloss.Center).
		Border(border, true).
		AlignVertical(lipgloss.Center).
		MaxWidth(w - border.GetLeftSize())
	theme.Focused.Card = theme.Focused.Base
	theme.Focused.Card = theme.Focused.Card.Foreground(lipgloss.Color("#ff5555"))
	return theme
}

func GetQuestionsFromTopics(topic persistence.TopicName, topicToQa map[persistence.TopicName][]string) (questions string) {
	var questionsLineSb strings.Builder
	for _, question := range topicToQa[topic] {
		questionsLineSb.WriteString(question + "\n")
	}
	questions = questionsLineSb.String()
	return
}
