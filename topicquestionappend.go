package main

import (
	"os"
	persistence "primotibalt/checkTests/topicpersistence"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
)

type AppendQuestion struct {
	Questions   []Question
	LeftPanelVp viewport.Model
	QaForm      *huh.Form
	QuestionSet bool
	Topic       persistence.TopicName
}

var (
	questionInputDesc string = "Введите вопрос"
	answerInputDesc   string = "Введите ответ на вопрос"
	UserAborted       string = ""
)

func ChooseTopic(topics []persistence.TopicName, topicToQa map[persistence.TopicName][]string) (selectedTopic persistence.TopicName, ok bool) {
	width, height, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		panic(err)
	}
	options := make([]huh.Option[persistence.TopicName], len(topics))
	for ptr, topic := range topics {
		options[ptr] = huh.NewOption(string(topic), topic)
	}
	MoveCursorToTopLeft()

	err = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[persistence.TopicName]().
				Title("Выбери топик для добавления вопросов:").
				Options(options...).
				Value(&selectedTopic),
			huh.NewNote().
				DescriptionFunc(func() string {
					return GetQuestionsFromTopics(selectedTopic, topicToQa)
				}, &selectedTopic).
				Title(questionsPanelTitle),
		),
	).WithWidth(width - 5).WithHeight(height - Padding).
		Run()
	if err != nil {
		if err != huh.ErrUserAborted {
			panic(err)
		} else {
			MoveCursorToTopLeft()
			return persistence.TopicName(UserAborted), false
		}
	}

	return selectedTopic, true
}

func RunTopicQuestionAppend(selectedTopic persistence.TopicName) {
	questions := []Question{}
	qaPairs := persistence.TopicQuestions(selectedTopic)
	for _, pair := range qaPairs {
		question, answer, ok := persistence.ParseQaLine(pair)
		if !ok {
			continue
		}

		questions = append(questions, Question{answer, question})
	}

	model := initializeQuestionAppendModel(questions, selectedTopic)
	program := tea.NewProgram(model, tea.WithInput(terminalInput))
	_, err := program.Run()
	if err != nil {
		panic(err)
	}
}

func (appendQuestionModel AppendQuestion) View() string {
	return lipgloss.JoinHorizontal(lipgloss.Left, appendQuestionModel.LeftPanelVp.View(), appendQuestionModel.QaForm.View())
}

func (appendQuestionModel AppendQuestion) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return appendQuestionModel, tea.Batch(tea.ClearScreen, tea.Quit)
		case tea.KeyEnter:
			// shift+enter arrives as alt+enter, which huh's text field already
			// binds to "new line" — let it through instead of moving on.
			if msg.Alt {
				break
			}

			if appendQuestionModel.QuestionSet {
				answer := appendQuestionModel.QaForm.GetFocusedField().GetValue()
				appendQuestionModel.QaForm.PrevField()
				question := appendQuestionModel.QaForm.GetFocusedField().GetValue()
				appendQuestionModel.Questions = append(appendQuestionModel.Questions, Question{Text: question.(string), Answer: answer.(string)})
				appendQuestionModel.QuestionSet = false
				appendQuestionModel.QaForm = getQaForm(appendQuestionModel.QaForm.Help().Width)
				persistence.AppendQuestionsToTopic(appendQuestionModel.Topic, map[string]string{question.(string): answer.(string)})
			} else {
				appendQuestionModel.QaForm.NextField()
				appendQuestionModel.QuestionSet = true
			}
		case tea.KeyCtrlZ:
			if appendQuestionModel.QuestionSet {
				appendQuestionModel.QaForm.PrevField()
				appendQuestionModel.QuestionSet = false
			}
		}
	}
	sb := strings.Builder{}
	for _, item := range appendQuestionModel.Questions {
		sb.WriteString("Q:" + item.PrintText(appendQuestionModel.LeftPanelVp.Width-2) + "\n")
		sb.WriteString("A:" + item.PrintAnswer(appendQuestionModel.LeftPanelVp.Width-2) + "\n\n")
	}
	appendQuestionModel.LeftPanelVp.SetContent(sb.String())
	appendQuestionModel.QaForm.Update(msg)
	appendQuestionModel.LeftPanelVp.Update(msg)

	return appendQuestionModel, nil
}

func (appendQuestionModel AppendQuestion) Init() tea.Cmd {
	return tea.ClearScreen
}

func initializeQuestionAppendModel(questions []Question, selectedTopic persistence.TopicName) (appendQuestionModel AppendQuestion) {
	width, height, termSizeErr := term.GetSize(os.Stdout.Fd())
	rightPanelWidth := width * 3 / 7
	leftPanelWidth := width - rightPanelWidth
	if termSizeErr != nil {
		width = 80
	}

	appendQuestionModel = AppendQuestion{
		LeftPanelVp: viewport.New(leftPanelWidth, height),
		Questions:   questions,
		QaForm:      getQaForm(rightPanelWidth),
		Topic:       selectedTopic,
	}

	return
}

func getQaForm(rightPanelWidth int) *huh.Form {
	questionInput := huh.NewText().Description(questionInputDesc)
	questionInput.Focus()
	answerInput := huh.NewText().Description(answerInputDesc)
	return huh.NewForm(huh.NewGroup(
		questionInput,
		answerInput,
	)).WithWidth(rightPanelWidth)
}
