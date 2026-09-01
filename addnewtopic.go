package main

import (
	"os"
	"strings"

	persistence "primotibalt/checkTests/topicpersistence"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
)

type AddNewTopicModel struct {
	LeftSidePanel   viewport.Model
	AddNewTopicForm *huh.Form
	Topics          []persistence.TopicName
}

func (m AddNewTopicModel) View() string {
	return lipgloss.JoinHorizontal(
		lipgloss.Left, m.LeftSidePanel.View(), m.AddNewTopicForm.View())
}

func (m AddNewTopicModel) Init() tea.Cmd {
	return tea.ClearScreen
}

func (m AddNewTopicModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		lspCmd  tea.Cmd
		antfCmd tea.Cmd
	)
	m.LeftSidePanel, lspCmd = m.LeftSidePanel.Update(msg)
	_, antfCmd = m.AddNewTopicForm.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			topicName := persistence.TopicName(m.AddNewTopicForm.GetFocusedField().GetValue().(string))
			if len(topicName) != 0 {
				_, err := persistence.AddNewTopic(topicName)
				if err != nil {
					panic(err)
				}
				m.Topics = append(m.Topics, topicName)

				var sb strings.Builder
				for _, topic := range m.Topics {
					sb.WriteString(string(topic) + "\n\n")
				}

				m.LeftSidePanel.SetContent(sb.String())
				return m, tea.Quit
			}
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}
	}

	return m, tea.Batch(lspCmd, antfCmd)
}

func AddNewTopic(topics []persistence.TopicName) {
	_, err := tea.NewProgram(initializeAddNewTopicModel(topics)).Run()
	if err != nil {
		panic(err)
	}
	MoveCursorToTopLeft()
}

func initializeAddNewTopicModel(topics []persistence.TopicName) AddNewTopicModel {
	width, height, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		panic(err)
	}
	leftSidePanelWidth := width * 4 / 7
	rightSidePanelWidth := width - leftSidePanelWidth

	addNewTopicTextInput := huh.NewText()
	addNewTopicTextInput.Focus()
	var sb strings.Builder
	for _, topic := range topics {
		sb.WriteString(string(topic) + "\n\n")
	}
	leftSidePanel := viewport.New(leftSidePanelWidth, height)
	leftSidePanel.SetContent(sb.String())

	model := AddNewTopicModel{
		LeftSidePanel: leftSidePanel,
		AddNewTopicForm: huh.NewForm(
			huh.NewGroup(
				addNewTopicTextInput.
					Description("Введи имя нового топика. Имя должно быть уникальным."))).
			WithWidth(rightSidePanelWidth),
		Topics: topics,
	}

	return model
}
