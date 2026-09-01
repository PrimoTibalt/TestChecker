package topicpersistence

import (
	"strings"
	"testing"
)

func TestQaLineRoundTripsSnippets(t *testing.T) {
	question := "Как выглядит main?"
	answer := "func main() {\n\tfmt.Println(\"hi\")\n}"

	line := FormatQaLine(question, answer)
	if strings.Count(line, "\n") != 1 || !strings.HasSuffix(line, "\n") {
		t.Fatalf("a pair must occupy exactly one line, got %q", line)
	}

	gotQuestion, gotAnswer, ok := ParseQaLine(strings.TrimSuffix(line, "\n"))
	if !ok {
		t.Fatal("ParseQaLine did not recognise the line it was given")
	}
	if gotQuestion != question {
		t.Errorf("question: got %q, want %q", gotQuestion, question)
	}
	if gotAnswer != answer {
		t.Errorf("answer: got %q, want %q", gotAnswer, answer)
	}
}

func TestParseQaLineReadsPreExistingLines(t *testing.T) {
	question, answer, ok := ParseQaLine("вопрос" + DelimeterQuestionAnswer + "ответ")
	if !ok || question != "вопрос" || answer != "ответ" {
		t.Errorf("got (%q, %q, %v), want (%q, %q, true)", question, answer, ok, "вопрос", "ответ")
	}
}

func TestParseQaLineRejectsLinesWithoutAPair(t *testing.T) {
	for _, line := range []string{"", "просто текст"} {
		if _, _, ok := ParseQaLine(line); ok {
			t.Errorf("ParseQaLine(%q) accepted a line holding no pair", line)
		}
	}
}

func TestParseQaLineKeepsDelimeterInsideAnswer(t *testing.T) {
	line := strings.TrimSuffix(FormatQaLine("q", "a"+DelimeterQuestionAnswer+"b"), "\n")
	_, answer, _ := ParseQaLine(line)
	if answer != "a"+DelimeterQuestionAnswer+"b" {
		t.Errorf("got %q, want %q", answer, "a"+DelimeterQuestionAnswer+"b")
	}
}
