package main

import (
	"testing"
)

func TestIgnoreIndentationLeavesSnippetsAtDistanceZero(t *testing.T) {
	expected := "func main() {\n\tfmt.Println(\"hi\")\n}"
	typedWithSpaces := "func main() {\n    fmt.Println(\"hi\")\n}"
	typedFlat := "func main() {\nfmt.Println(\"hi\")\n}\n\n"

	for _, actual := range []string{expected, typedWithSpaces, typedFlat} {
		if distance := Ld(IgnoreIndentation(actual), IgnoreIndentation(expected)); distance != 0 {
			t.Errorf("Ld(%q) = %d, want 0", actual, distance)
		}
	}
}

func TestIgnoreIndentationKeepsRealDifferences(t *testing.T) {
	expected := "a\n\tb"
	actual := "a\n\tc"

	if distance := Ld(IgnoreIndentation(actual), IgnoreIndentation(expected)); distance != 1 {
		t.Errorf("Ld = %d, want 1", distance)
	}
}

func TestIgnoreIndentationKeepsLineBreaks(t *testing.T) {
	if got := IgnoreIndentation("  a  \n\n\t b\t"); got != "a\nb" {
		t.Errorf("got %q, want %q", got, "a\nb")
	}
}
