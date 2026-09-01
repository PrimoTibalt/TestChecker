package main

import (
	"testing"
)

func TestTranslateShiftEnterRewritesCsiU(t *testing.T) {
	cases := map[string]struct {
		in   string
		rest string
		out  string
	}{
		"shift+enter becomes alt+enter":     {"\x1b[13;2u", "", "\x1b\r"},
		"alt+shift+enter becomes alt+enter": {"\x1b[13;4u", "", "\x1b\r"},
		"xterm encoding is understood too":  {"\x1b[27;2;13~", "", "\x1b\r"},
		"plain enter is left alone":         {"\r", "", "\r"},
		"escape is left alone":              {"\x1b", "", "\x1b"},
		"ctrl+c is left alone":              {"\x03", "", "\x03"},
		"enter without shift is left alone": {"\x1b[13;5u", "", "\x1b[13;5u"},
		"other csi sequences pass through":  {"\x1b[1;5A", "", "\x1b[1;5A"},
		"text around a sequence survives":   {"ab\x1b[13;2ucd", "", "ab\x1b\rcd"},
		"a split sequence is held back":     {"ab\x1b[13;", "\x1b[13;", "ab"},
		"a split escape is not held back":   {"ab\x1b", "", "ab\x1b"},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			rest, out := translateShiftEnter([]byte(testCase.in), nil)
			if string(rest) != testCase.rest {
				t.Errorf("rest: got %q, want %q", rest, testCase.rest)
			}
			if string(out) != testCase.out {
				t.Errorf("out: got %q, want %q", out, testCase.out)
			}
		})
	}
}

func TestTranslateShiftEnterAcrossReads(t *testing.T) {
	var buffer, pending []byte

	buffer, pending = translateShiftEnter(append(buffer, "hi\x1b[13"...), pending)
	if string(pending) != "hi" {
		t.Fatalf("first read: got %q, want %q", pending, "hi")
	}

	buffer, pending = translateShiftEnter(append(buffer, ";2u!"...), pending)
	if string(buffer) != "" {
		t.Errorf("buffer should be drained, got %q", buffer)
	}
	if string(pending) != "hi\x1b\r!" {
		t.Errorf("second read: got %q, want %q", pending, "hi\x1b\r!")
	}
}
