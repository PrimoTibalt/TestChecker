package main

import (
	"fmt"
	"slices"
	"strings"

	persistence "primotibalt/checkTests/topicpersistence"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// ChooseTopicsModel is the multi-select of topics to be tested on, with the
// questions of the topic under the cursor listed underneath it.
type ChooseTopicsModel struct {
	Form          *huh.Form
	Topics        *huh.MultiSelect[persistence.TopicName]
	QuestionsNote *huh.Note
	TopicToQa     map[persistence.TopicName][]string
	HoveredTopic  persistence.TopicName
}

func ChooseTopicsForTest(topics []persistence.TopicName, topicToQa map[persistence.TopicName][]string) (selectedTopics []persistence.TopicName) {
	width, height := terminalSize()
	model := newChooseTopicsModel(topics, topicToQa, &selectedTopics, width-5, height-Padding)

	finished, err := tea.NewProgram(model, tea.WithAltScreen(), tea.WithInput(terminalInput)).Run()
	if err != nil {
		panic(err)
	}

	if finished.(ChooseTopicsModel).Form.State == huh.StateAborted {
		return []persistence.TopicName{}
	}

	return
}

func (m ChooseTopicsModel) Init() tea.Cmd {
	return tea.Batch(tea.ClearScreen, m.Form.Init())
}

func (m ChooseTopicsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	form, cmd := m.Form.Update(msg)
	m.Form = form.(*huh.Form)

	if m.showQuestionsOfHoveredTopic() {
		rebuildFormView(m.Form)
	}

	// The form decides itself what submits and what aborts; we only follow it
	// out of the program once it is done.
	if m.Form.State != huh.StateNormal {
		return m, tea.Batch(cmd, tea.Quit)
	}

	return m, cmd
}

func (m ChooseTopicsModel) View() string {
	return m.Form.View()
}

func newChooseTopicsModel(
	topics []persistence.TopicName,
	topicToQa map[persistence.TopicName][]string,
	selectedTopics *[]persistence.TopicName,
	width, height int,
) ChooseTopicsModel {
	options := make([]huh.Option[persistence.TopicName], len(topics))
	for ptr, topic := range topics {
		options[ptr] = huh.NewOption(string(topic), topic)
	}

	topicsSelect := huh.NewMultiSelect[persistence.TopicName]().
		Title("Выбери топик(и) для теста:").
		Options(options...).
		Value(selectedTopics)
	questionsNote := newQuestionsNote()

	model := ChooseTopicsModel{
		Form: huh.NewForm(huh.NewGroup(topicsSelect, questionsNote)).
			WithWidth(width).
			WithHeight(height),
		Topics:        topicsSelect,
		QuestionsNote: questionsNote,
		TopicToQa:     topicToQa,
	}
	// Fill the panel in before the first frame, which is drawn before any
	// message reaches the model.
	model.showQuestionsOfHoveredTopic()
	rebuildFormView(model.Form)

	return model
}

// showQuestionsOfHoveredTopic points the panel at the topic under the cursor.
// A multi-select binds only what is checked, so the cursor has to be read off
// the field itself. Reports whether the panel changed.
func (m *ChooseTopicsModel) showQuestionsOfHoveredTopic() bool {
	hovered, ok := m.Topics.Hovered()
	if !ok || hovered == m.HoveredTopic {
		return false
	}

	m.HoveredTopic = hovered
	m.QuestionsNote.Description(GetQuestionsFromTopics(hovered, m.TopicToQa))

	return true
}

func RunKnowledgeTest(selectedTopics []persistence.TopicName) {
	questions := []Question{}
	for _, topic := range selectedTopics {
		qaPairs := persistence.TopicQuestions(topic)
		for _, pair := range qaPairs {
			question, answer, ok := persistence.ParseQaLine(pair)
			if !ok {
				continue
			}

			questions = append(questions, Question{answer, question})
		}
	}

	model, err := initializeModel(questions)
	if err != nil {
		panic(err)
	}

	program := tea.NewProgram(model, tea.WithAltScreen(), tea.WithInput(terminalInput))
	_, err = program.Run()
	if err != nil {
		panic(err)
	}
}

func (m TestCheck) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, tea.ClearScreen)
}

func (m TestCheck) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		taCmd       tea.Cmd
		vpCmd       tea.Cmd
		vpFailedCmd tea.Cmd
	)

	m.textarea, taCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)
	m.vpFailed, vpFailedCmd = m.vpFailed.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			// shift+enter reaches us as alt+enter; the textarea has already
			// turned it into a line break, so there is nothing to submit.
			if msg.Alt {
				break
			}

			if len(m.Questions) < 1 {
				switch strings.Trim(m.textarea.Value(), " \n") {
				case "r", "R", "reset":
					m, err := initializeModel(slices.Concat(m.SuccessQuestions, m.FailedQuestions))
					if err != nil {
						panic(err)
					}
					return m, tea.Batch(taCmd, vpCmd, tea.ClearScreen)
				default:
					return m, tea.Quit
				}
			}

			distance := m.computeDistanceBetweenInputAndAnswer()
			m.LastQuestionSuccess = m.IsInputAndAnswerEqual(distance)
			if m.LastQuestionSuccess {
				m.SuccessQuestions = append(m.SuccessQuestions, *m.CurrentQuestion)
			} else {
				m.FailedQuestions = append(m.FailedQuestions, *m.CurrentQuestion)
				m.prepareFailedVpContent(strings.Trim(m.textarea.Value(), " \n\r"))
			}

			m.textarea.Reset()
			m.prepareNextQuestion()
		}
	}

	m.growTextareaToFitInput()

	return m, tea.Batch(taCmd, vpCmd, vpFailedCmd)
}

func (m TestCheck) View() string {
	const delimeter = "\n"

	if len(m.Questions) > 0 {
		return lipgloss.JoinHorizontal(lipgloss.Left,
			lipgloss.JoinVertical(lipgloss.Left, m.viewport.View(), m.textarea.View()),
			m.vpFailed.View())
	} else {
		return fmt.Sprintf("%s%s%s%s",
			lipgloss.JoinHorizontal(lipgloss.Left,
				lipgloss.JoinVertical(lipgloss.Left, m.viewport.View(), m.textarea.View()),
				m.vpFailed.View()),
			delimeter,
			"Нажми r(reset) для продолжения. Любую другую для возвращения к списку.",
			delimeter)
	}
}
