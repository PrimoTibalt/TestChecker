package main

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
)

type Question struct {
	Answer string
	Text   string
}

func (q *Question) PrintAnswer(consoleWidth int) string {
	return printWithoutBreaksInWords(q.Answer, consoleWidth)
}

func (q *Question) PrintText(consoleWidth int) string {
	return printWithoutBreaksInWords(q.Text, consoleWidth)
}

// printWithoutBreaksInWords wraps each line on its own, so the line breaks a
// code snippet was written with survive into the panel.
func printWithoutBreaksInWords(text string, consoleWidth int) (splitText string) {
	lines := []string{}
	for line := range strings.SplitSeq(text, "\n") {
		lines = append(lines, wrapWithoutBreaksInWords(line, consoleWidth))
	}

	return strings.Join(lines, "\n")
}

func wrapWithoutBreaksInWords(text string, consoleWidth int) (splitText string) {
	var builder strings.Builder
	var counter int
	var index int
	var lastWhitespaceIdx int
	runes := []rune(text)
	for counter < len(runes) {
		if index == consoleWidth {
			originalCounter := counter
			counter = lastWhitespaceIdx

			if counter == 0 || index == 0 {
				counter = originalCounter - 1
			}

			index = 0
			builder.WriteRune('\n')
		}

		if runes[counter] == ' ' || counter == len(runes)-1 {
			if counter == len(runes)-1 {
				counter++
			}

			switch {
			case lastWhitespaceIdx == 0:
				builder.WriteString(string(runes[0 : counter+1]))
			case lastWhitespaceIdx+1 < counter && counter != len(runes):
				builder.WriteString(string(runes[lastWhitespaceIdx+1 : counter+1]))
			case lastWhitespaceIdx+1 < counter:
				builder.WriteString(string(runes[lastWhitespaceIdx+1 : counter]))
			}
			lastWhitespaceIdx = counter
		}
		index++
		counter++
	}

	splitText = builder.String()
	return
}

type TestCheck struct {
	viewport            viewport.Model
	textarea            textarea.Model
	vpFailed            viewport.Model
	Questions           []Question
	SuccessQuestions    []Question
	FailedQuestions     []Question
	CurrentQuestion     *Question
	LastQuestionSuccess bool
}

const (
	initialTextareaHeight    = 2
	smallAnswerLen           = 10
	mediumAnswerLen          = 15
	bigAnswerLen             = 25
	paragraphAnswerLen       = 100
	poemAnswerLength         = 250
	dissertationAnswerLength = 1000
)

func (m *TestCheck) setContentWhenNoMoreQuestions() {
	sb := strings.Builder{}
	if len(m.FailedQuestions) > 0 {
		fmt.Fprintf(&sb, "Неправильно ответил на %d вопросов из %d.\n",
			len(m.FailedQuestions),
			len(m.FailedQuestions)+len(m.SuccessQuestions))
		sb.WriteString("Заваленные вопросы:\n")
		for _, question := range m.FailedQuestions {
			fmt.Fprintf(&sb, "Вопрос: %s\nОтвет: %s\n", question.Text, question.Answer)
		}
	} else {
		sb.WriteString("Вы не завалили ни одного вопроса. Молодец!\n")
	}

	resultString := sb.String()
	m.viewport.SetContent(resultString)
	m.viewport.Height = strings.Count(resultString, "\n") + 3
	m.viewport.Style.Padding(1)
}

func (m *TestCheck) computeDistanceBetweenInputAndAnswer() (distance int) {
	distance = Ld(IgnoreIndentation(m.textarea.Value()), IgnoreIndentation(m.CurrentQuestion.Answer))
	return
}

func (m *TestCheck) IsInputAndAnswerEqual(distance int) (result bool) {
	answerLen := utf8.RuneCountInString(IgnoreIndentation(m.CurrentQuestion.Answer))
	if answerLen <= smallAnswerLen {
		result = distance <= 0
	} else if answerLen <= mediumAnswerLen {
		result = distance <= 2
	} else if answerLen <= bigAnswerLen {
		result = distance <= 4
	} else if answerLen <= paragraphAnswerLen {
		result = distance <= 8
	} else if answerLen <= poemAnswerLength {
		result = distance <= 16
	} else if answerLen <= dissertationAnswerLength {
		result = distance <= 32
	} else {
		result = distance < 100
	}

	return
}

func initializeModel(questions []Question) (testCheckModel TestCheck, err error) {
	width, height, termSizeErr := term.GetSize(os.Stdout.Fd())
	if termSizeErr != nil {
		panic(termSizeErr)
	}
	rightPanelWidth := width * 3 / 7
	leftPanelWidth := width - rightPanelWidth
	if len(questions) == 0 {
		err = errors.New("выбранный файл не содержит вопросов попробуйте добавить новые")
		return TestCheck{}, err
	}
	currentQuestion := questions[rand.Intn(len(questions))]

	taModel := textarea.New()
	taModel.Focus()
	taModel.SetHeight(initialTextareaHeight)
	taModel.KeyMap = textarea.DefaultKeyMap
	// Enter submits the answer, so a line break needs a key of its own.
	taModel.KeyMap.InsertNewline = key.NewBinding(
		key.WithKeys(newLineKeys...),
		key.WithHelp("shift+enter", "новая строка"))
	taModel.Placeholder = defaultTaPlaceholder
	taModel.FocusedStyle.CursorLine = lipgloss.NewStyle()
	taModel.ShowLineNumbers = false
	taModel.SetWidth(leftPanelWidth)
	taModel.FocusedStyle.Base = lipgloss.NewStyle().Border(lipgloss.MarkdownBorder(), false, false, false, true)

	vpModel := viewport.New(leftPanelWidth, 8)
	vpModel.KeyMap = viewport.KeyMap{}
	vpModel.KeyMap.Down = key.NewBinding(key.WithKeys("down"))
	vpModel.KeyMap.Up = key.NewBinding(key.WithKeys("up"))
	vpModel.Style = vpModel.Style.Border(
		lipgloss.DoubleBorder(),
		true, true,
	).
		BorderForeground(lipgloss.Color("6")).
		Foreground(lipgloss.Color("180"))

	vpModel.SetContent(currentQuestion.PrintText(vpModel.Width))

	vpFailedModel := viewport.New(rightPanelWidth, height-2)
	vpFailedModel.KeyMap = viewport.KeyMap{}
	vpFailedModel.KeyMap.Down = key.NewBinding(key.WithKeys("down"))
	vpFailedModel.KeyMap.Up = key.NewBinding(key.WithKeys("up"))
	vpFailedModel.Style = vpFailedModel.Style.Border(
		lipgloss.NormalBorder(),
		true,
	).
		BorderForeground(lipgloss.Color("52"))

	testCheckModel = TestCheck{
		vpModel,
		taModel,
		vpFailedModel,
		questions,
		[]Question{},
		[]Question{},
		&currentQuestion,
		true,
	}
	return
}

// growTextareaToFitInput keeps every line the answer has so far on screen,
// whether the lines came from wrapping or from shift+enter.
func (m *TestCheck) growTextareaToFitInput() {
	needed := max(m.textarea.Height(), m.textarea.LineCount())
	if m.textarea.Length() >= m.textarea.Width()*m.textarea.Height() {
		needed++
	}

	if needed == m.textarea.Height() {
		return
	}

	currentContent := m.textarea.Value()
	m.textarea.SetHeight(needed)
	m.textarea.SetValue(currentContent)
}

func (m *TestCheck) prepareNextQuestion() {
	m.textarea.SetHeight(initialTextareaHeight)

	questionIndex := slices.Index(m.Questions, *m.CurrentQuestion)
	m.Questions = slices.Delete(m.Questions, questionIndex, questionIndex+1)

	questionsToAskCount := len(m.Questions)
	if questionsToAskCount > 0 {
		selectedQuestionIndex := rand.Intn(questionsToAskCount)
		m.CurrentQuestion = &m.Questions[selectedQuestionIndex]
		resultString := m.CurrentQuestion.PrintText(m.viewport.Width)
		m.viewport.SetContent(resultString)
		m.viewport.Height = strings.Count(resultString, "\n") + 3
	} else {
		m.setContentWhenNoMoreQuestions()
	}
}

func (m *TestCheck) prepareFailedVpContent(yourAnswer string) {
	var builder strings.Builder
	for l := range strings.SplitSeq(m.vpFailed.View(), "\n") {
		if l != "" {
			builder.WriteString(l + "\n")
		}
	}

	m.vpFailed.SetContent(
		printWithoutBreaksInWords("V: "+m.CurrentQuestion.Answer, m.vpFailed.Width-1) + "\n" +
			printWithoutBreaksInWords("X: "+yourAnswer, m.vpFailed.Width-1) + "\n" +
			builder.String(),
	)
}
