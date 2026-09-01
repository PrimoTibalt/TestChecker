package main

import (
	"os"
	persistence "primotibalt/checkTests/topicpersistence"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/x/term"
)

type RemoveTopicModel struct {
	RemoveTopicForm *huh.Form
	UserAborted     bool
}

func (m RemoveTopicModel) Init() tea.Cmd {
	return tea.ClearScreen
}

func (m RemoveTopicModel) Update(msg tea.Msg) (model tea.Model, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.UserAborted = true
			return m, tea.Batch(tea.ClearScreen, tea.Quit)
		case tea.KeyEnter:
			return m, tea.Batch(tea.ClearScreen, tea.Quit)
		}
	}

	_, cmd = m.RemoveTopicForm.Update(msg)
	model = m

	return
}

func (m RemoveTopicModel) View() string {
	return m.RemoveTopicForm.View()
}

func RemoveTopic(topics []persistence.TopicName, topicToQa map[persistence.TopicName][]string) {
	width, height, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		panic(err)
	}

	topicsOptions := make([]huh.Option[persistence.TopicName], len(topics))
	for ptr, topic := range topics {
		topicsOptions[ptr] = huh.NewOption(string(topic), topic)
	}

	var topicToRemove persistence.TopicName
	selectTopicPanel := huh.NewSelect[persistence.TopicName]().Options(
		topicsOptions...).
		Value(&topicToRemove)
	questionsPanel := huh.NewNote().DescriptionFunc(func() string {
		return GetQuestionsFromTopics(topicToRemove, topicToQa)
	}, &topicToRemove)

	selectTopicGroup := huh.NewGroup(selectTopicPanel, questionsPanel).WithWidth(width).WithHeight(height)
	form := huh.NewForm(selectTopicGroup).WithWidth(width).WithHeight(height - Padding)
	model := RemoveTopicModel{
		form,
		false,
	}
	m, err := tea.NewProgram(model).Run()
	if err != nil {
		panic(err)
	}

	MoveCursorToTopLeft()
	if m.(RemoveTopicModel).UserAborted {
		return
	}

	if len(topicToRemove) > 0 {
		var userConfirmed bool

		huh.NewConfirm().Value(&userConfirmed).
			Affirmative("Удалить").Negative("Отмена").
			Description(GetQuestionsFromTopics(topicToRemove, topicToQa)).
			Title("Вы уверены что хотите удалить топик \"" + string(topicToRemove) + "\"?").
			WithTheme(MakeThemeCenterInnerElement(huh.ThemeCharm())).
			Run()

		MoveCursorToTopLeft()
		if userConfirmed {
			persistence.RemoveTopic(topicToRemove)
		}
	}
}
