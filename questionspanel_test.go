package main

import (
	persistence "primotibalt/checkTests/topicpersistence"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

var panelTopicToQa = map[persistence.TopicName][]string{
	"go":  {"что такое горутина", "что такое канал"},
	"sql": {"что такое индекс"},
}

func TestTestChooserShowsQuestionsOfHoveredTopic(t *testing.T) {
	topics := []persistence.TopicName{"go", "sql"}
	selectedTopics := []persistence.TopicName{}
	model := newChooseTopicsModel(topics, panelTopicToQa, &selectedTopics, 80, 20)
	model.Init()

	if view := model.View(); !strings.Contains(view, "что такое горутина") {
		t.Fatalf("questions of the first topic are not shown before any key press:\n%s", view)
	}

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	view := updated.View()
	if !strings.Contains(view, "что такое индекс") {
		t.Errorf("questions of the topic under the cursor are not shown:\n%s", view)
	}
	if strings.Contains(view, "что такое горутина") {
		t.Errorf("questions of the previously hovered topic are still shown:\n%s", view)
	}
}

func TestEditTopicShowsQuestionsOfHoveredTopic(t *testing.T) {
	model := initializeEditTopicModel(map[persistence.TopicName][]QaPair{
		"go":  {{"что такое горутина", "поток"}, {"что такое канал", "труба"}},
		"sql": {{"что такое индекс", "дерево"}},
	})

	hovered := persistence.TopicName(selectedValue)
	for _, question := range panelTopicToQa[hovered] {
		if !strings.Contains(model.Select.View(), question) {
			t.Fatalf("question %q of the hovered topic %q is not shown:\n%s", question, hovered, model.Select.View())
		}
	}

	// Walking the list has to keep the panel on the topic under the cursor.
	model.processInputInSelectingMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	hovered = persistence.TopicName(selectedValue)
	for _, question := range panelTopicToQa[hovered] {
		if !strings.Contains(model.Select.View(), question) {
			t.Errorf("question %q of the hovered topic %q is not shown after moving:\n%s", question, hovered, model.Select.View())
		}
	}
}

func TestEditTopicDropsTheQuestionsPanelWhenPickingAQuestion(t *testing.T) {
	model := initializeEditTopicModel(map[persistence.TopicName][]QaPair{
		"go": {{"что такое горутина", "поток"}},
	})

	model.processInputInSelectingMode(tea.KeyMsg{Type: tea.KeyEnter})

	if model.EditingPart != QuestionPart {
		t.Fatalf("enter on a topic did not move on to its questions, got %q", model.EditingPart)
	}
	if model.QuestionsNote != nil {
		t.Error("the questions panel is still attached while a question is being picked")
	}
}
