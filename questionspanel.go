package main

import (
	"github.com/charmbracelet/huh"
)

const questionsPanelTitle = "Вопросы"

// refreshViewMsg is inert for every field: it exists only to make a group run
// an update cycle. A huh group renders the view it cached during its last
// update, so a form whose fields were changed from the outside — or one that
// has not been updated at all yet — shows nothing until it gets a message.
type refreshViewMsg struct{}

// rebuildFormView refreshes what form renders after its fields changed outside
// of an update.
func rebuildFormView(form *huh.Form) {
	form.Update(refreshViewMsg{})
}

// newQuestionsNote builds the panel that lists the questions of the topic the
// cursor sits on, the way the topic picker of "add question" does.
func newQuestionsNote() *huh.Note {
	return huh.NewNote().Title(questionsPanelTitle)
}
