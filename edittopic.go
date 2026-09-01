package main

import (
	"errors"
	"os"
	"slices"
	"strings"

	persistence "primotibalt/checkTests/topicpersistence"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
)

type EditTopicModel struct {
	Input    *huh.Text
	Original string
	Select   *huh.Form

	EditingPart EditingPart
	IsEditing   bool

	Content          map[persistence.TopicName]map[string]string
	SelectedTopic    persistence.TopicName
	SelectedQuestion string

	// QuestionsNote lists the questions of the topic under the cursor while a
	// topic is being picked, and is nil in every other state.
	QuestionsNote *huh.Note
	HoveredTopic  persistence.TopicName
}

type EditingPart string

var (
	TopicPart     EditingPart = "Topic"
	QuestionPart  EditingPart = "Question"
	AnswerPart    EditingPart = "Answer"
	selectedValue string      = ""
)

func (m EditTopicModel) View() string {
	w, h, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		os.Exit(1)
	}
	if m.IsEditing {
		inputTheme := huh.ThemeBase()
		inputTheme.Focused.Base = lipgloss.NewStyle().
			PaddingLeft(Padding).
			PaddingRight(Padding).
			AlignHorizontal(lipgloss.Center).AlignVertical(lipgloss.Center).
			MaxWidth(w - border.GetLeftSize() - Padding).
			MaxHeight(h - border.GetTopSize() - Padding)
		inputTheme.Focused.Card = inputTheme.Focused.Base
		content := lipgloss.JoinVertical(
			lipgloss.Center,
			renderOriginal(m.Original, w-border.GetLeftSize()-border.GetRightSize()-Padding),
			m.Input.
				WithTheme(inputTheme).
				WithWidth(w-border.GetLeftSize()-border.GetRightSize()-Padding).
				WithHeight(h-border.GetTopSize()-border.GetBottomSize()-4*Padding).
				View())
		return borderStyle.Render(content)
	}

	return lipgloss.JoinVertical(
		lipgloss.Center,
		m.Select.WithWidth(w-Padding).
			WithHeight(h-Padding).
			View())
}

func (m EditTopicModel) Init() tea.Cmd {
	return tea.ClearScreen
}

func (m EditTopicModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
	}
	if !m.IsEditing {
		m.processInputInSelectingMode(msg)
		return m, nil
	}
	if m.IsEditing {
		m.processInputInEditingMode(msg)
		return m, nil
	}
	return m, nil
}

func (m *EditTopicModel) processInputInSelectingMode(msg tea.Msg) {
	if key, isKey := msg.(tea.KeyMsg); isKey {
		switch key.String() {
		case "tab", "enter":
			if m.EditingPart != AnswerPart {
				m.calculateNextState()
			}
		case "shift+tab", "esc":
			if m.EditingPart != TopicPart {
				m.calculatePrevState()
			}
		case "e":
			m.updateInputWithNewValue()
		}
	}

	teaModel, _ := m.Select.Update(msg)
	teaSelect, _ := teaModel.(*huh.Form)
	m.Select = teaSelect

	if m.showQuestionsOfHoveredTopic() {
		rebuildFormView(m.Select)
	}
}

// showQuestionsOfHoveredTopic points the questions panel at the topic the
// cursor moved onto. Reports whether the panel changed.
func (m *EditTopicModel) showQuestionsOfHoveredTopic() bool {
	if m.QuestionsNote == nil || persistence.TopicName(selectedValue) == m.HoveredTopic {
		return false
	}

	m.HoveredTopic = persistence.TopicName(selectedValue)
	m.QuestionsNote.Description(questionsOfTopic(m.Content[m.HoveredTopic]))

	return true
}

// questionsOfTopic lists a topic's questions one per line, in a stable order
// since the content is kept in a map.
func questionsOfTopic(content map[string]string) string {
	questions := make([]string, 0, len(content))
	for question := range content {
		questions = append(questions, asOneLine(question))
	}
	slices.Sort(questions)

	return strings.Join(questions, "\n")
}

func (m *EditTopicModel) processInputInEditingMode(msg tea.Msg) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" || msg.String() == "enter" {
			if msg.String() == "enter" {
				m.applyChanges()
				m.updateSelectWithOptions()
			}
			m.IsEditing = false
			if m.EditingPart == AnswerPart {
				m.calculatePrevState()
			}
			return
		}
	}
	input, _ := m.Input.Update(msg)
	inputText, ok := input.(*huh.Text)
	if ok {
		m.Input = inputText
	}
}

func (m *EditTopicModel) updateTopicContent(updatedTopicName string) {
	initialTopic := persistence.TopicName(m.Original)
	content := m.Content[initialTopic]
	delete(m.Content, initialTopic)
	m.Content[persistence.TopicName(updatedTopicName)] = content

	topics := []persistence.TopicName{}
	for topic := range m.Content {
		topics = append(topics, topic)
	}
	persistence.UpdateTopicNames(topics)
}

func (m *EditTopicModel) updateQuestionContent(updatedQuestion string) {
	initialQuestion := m.Original
	content := m.Content[m.SelectedTopic][initialQuestion]
	delete(m.Content[m.SelectedTopic], initialQuestion)
	m.Content[m.SelectedTopic][updatedQuestion] = content
	persistence.UpdateTopicFile(m.SelectedTopic, m.Content[m.SelectedTopic])
}

func (m *EditTopicModel) updateAnswerContent(updatedAnswer string) {
	m.Content[m.SelectedTopic][m.SelectedQuestion] = updatedAnswer
	persistence.UpdateTopicFile(m.SelectedTopic, m.Content[m.SelectedTopic])
}

func (m *EditTopicModel) noChangesInTopicOrDuplicateName(updatedValue string) bool {
	_, exists := m.Content[persistence.TopicName(updatedValue)]
	return exists
}

func (m *EditTopicModel) noChangesInQuestionOrDuplicateText(updatedValue string) bool {
	_, exists := m.Content[m.SelectedTopic][updatedValue]
	return exists
}

func (m *EditTopicModel) applyChanges() {
	updatedValue := strings.Trim(selectedValue, "\n\t ")
	// Questions and answers may hold code snippets and so span several lines;
	// a topic name is a file name, so it may not.
	multiLineTopicName := m.EditingPart == TopicPart && strings.Contains(updatedValue, "\n")
	if updatedValue == "" || multiLineTopicName {
		var errorMsg string
		switch {
		case updatedValue == "":
			errorMsg = "Новое значение не содержит никаких символов, так нельзя"
		case multiLineTopicName:
			errorMsg = "Нельзя переходить на новую строку в имени топика"
		}
		displayErrorNotification(errorMsg)
		return
	}
	switch m.EditingPart {
	case TopicPart:
		if m.noChangesInTopicOrDuplicateName(updatedValue) {
			return
		}
		m.updateTopicContent(updatedValue)
	case QuestionPart:
		if m.noChangesInQuestionOrDuplicateText(updatedValue) {
			return
		}
		m.updateQuestionContent(updatedValue)
	case AnswerPart:
		m.updateAnswerContent(updatedValue)
	}
}

func (m *EditTopicModel) updateSelectWithOptions() {
	partSelect := huh.NewSelect[string]().
		Options(func() []huh.Option[string] {
			options := []huh.Option[string]{}
			switch m.EditingPart {
			case TopicPart:
				for topic := range m.Content {
					options = append(options, huh.NewOption(string(topic), string(topic)))
				}
			case QuestionPart:
				for question := range m.Content[m.SelectedTopic] {
					options = append(options, huh.NewOption(asOneLine(question), question))
				}
			case AnswerPart:
				for _, answer := range m.Content[m.SelectedTopic] {
					options = append(options, huh.NewOption(asOneLine(answer), answer))
				}
			default:
				panic(errors.New("неизвестное состояние для селекта"))
			}
			return options
		}()...).
		Value(&selectedValue).
		Title(getTitleForSelectByState(m.EditingPart))

	fields := []huh.Field{partSelect}
	// A question is easier to track down when the topic list shows what each
	// topic holds, the way the topic picker of "add question" does.
	m.QuestionsNote = nil
	m.HoveredTopic = ""
	if m.EditingPart == TopicPart {
		m.QuestionsNote = newQuestionsNote()
		fields = append(fields, m.QuestionsNote)
	}

	// The form is sized here as well as in View because a group renders the
	// view it cached, and a note laid out at no width at all drops its text.
	width, height := terminalSize()
	m.Select = huh.NewForm(huh.NewGroup(fields...)).
		WithWidth(width - Padding).
		WithHeight(height - Padding)
	m.showQuestionsOfHoveredTopic()
	// A brand new form has never been updated, so it has nothing to render yet.
	rebuildFormView(m.Select)
}

func (m *EditTopicModel) updateInputWithNewValue() {
	m.IsEditing = true
	var note string
	switch m.EditingPart {
	case TopicPart:
		m.SelectedTopic = persistence.TopicName(selectedValue)
		note = selectedValue
	case QuestionPart:
		m.SelectedQuestion = selectedValue
		note = selectedValue
	case AnswerPart:
		note = m.Content[m.SelectedTopic][m.SelectedQuestion]
	}
	m.Original = note
	m.Input = newSnippetInput()
}

func (m *EditTopicModel) calculatePrevState() {
	switch m.EditingPart {
	case TopicPart:
	case QuestionPart:
		m.EditingPart = TopicPart
		m.SelectedTopic = ""
	case AnswerPart:
		m.EditingPart = QuestionPart
		m.IsEditing = false
		m.SelectedQuestion = ""
	}
	m.updateSelectWithOptions()
}

func (m *EditTopicModel) calculateNextState() {
	switch m.EditingPart {
	case TopicPart:
		m.EditingPart = QuestionPart
		m.SelectedTopic = persistence.TopicName(selectedValue)
	case QuestionPart:
		m.EditingPart = AnswerPart
		m.SelectedQuestion = selectedValue
	case AnswerPart:
	}
	if m.EditingPart == AnswerPart {
		m.IsEditing = true
		m.SelectedQuestion = selectedValue
		note := m.Content[m.SelectedTopic][m.SelectedQuestion]
		selectedValue = note
		m.Original = note
		m.Input = newSnippetInput()
	} else {
		m.updateSelectWithOptions()
	}
}

func EditTopic(topicToQa map[persistence.TopicName][]QaPair) {
	model := initializeEditTopicModel(topicToQa)
	_, err := tea.NewProgram(model, tea.WithAltScreen(), tea.WithInput(terminalInput)).Run()
	if err != nil {
		panic(err)
	}
}

// asOneLine keeps a multi-line snippet from pushing the rows of a select apart:
// only the label is folded, the option still carries the real value.
func asOneLine(text string) string {
	return strings.ReplaceAll(text, "\n", " ⏎ ")
}

// newSnippetInput builds the editor for a question or an answer. The field runs
// on its own rather than inside a form, so nothing hands it a keymap — without
// one there is no binding for a line break at all.
func newSnippetInput() *huh.Text {
	keymap := huh.NewDefaultKeyMap()
	keymap.Text.NewLine.SetKeys(newLineKeys...)

	input := huh.NewText().Value(&selectedValue)
	input.WithKeyMap(keymap)
	input.Focus()
	return input
}

func initializeEditTopicModel(topicToQa map[persistence.TopicName][]QaPair) EditTopicModel {
	m := EditTopicModel{
		EditingPart: TopicPart,
	}
	content := map[persistence.TopicName]map[string]string{}
	for topic, v := range topicToQa {
		content[topic] = map[string]string{}
		for _, qaPair := range v {
			content[topic][qaPair.Question] = qaPair.Answer
		}
	}
	m.Content = content
	m.updateSelectWithOptions()
	return m
}

func getTitleForSelectByState(editingPart EditingPart) string {
	switch editingPart {
	case TopicPart:
		return "Выберите топик для редактирования"
	case QuestionPart:
		return "Выберите вопрос для редактирования"
	case AnswerPart:
		return "Редактируйте ответ на вопрос"
	default:
		panic(errors.New("непредвиденное состояние селекта"))
	}
}

func displayErrorNotification(errorMsg string) {
	huh.NewForm(huh.NewGroup(
		huh.NewNote().
			Title(errorMsg).
			WithTheme(MakeThemeCenterInnerElement(huh.ThemeBase())),
	)).
		WithProgramOptions(tea.WithAltScreen()).
		Run()
}

func renderOriginal(original string, maxWidth int) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7571F9")).
		PaddingTop(1).
		Inline(true).
		MaxWidth(maxWidth).
		Render(original)
}
