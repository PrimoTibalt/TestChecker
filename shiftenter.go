package main

import (
	"bytes"
	"os"
	"strconv"
	"strings"
)

// Shift+enter has no encoding of its own in a plain terminal: it arrives as the
// very same carriage return enter sends, which is why the two used to be
// indistinguishable here. Terminals can be told to send it as the CSI-u
// sequence `ESC [ 13 ; 2 u` instead, but tmux only forwards such extended keys
// to an application that asked for them, and bubbletea v1 neither asks nor
// knows how to parse CSI u.
//
// So this file does both halves. requestExtendedKeys asks for xterm's
// modifyOtherKeys level 1 — level 2 would also re-encode esc and ctrl+c, which
// every model here matches on — and ShiftEnterInput rewrites the sequence into
// ESC+CR, i.e. alt+enter, which bubbles' textarea and huh's text field already
// know as "insert newline".
const (
	requestExtendedKeysSeq = "\x1b[>4;1m"
	releaseExtendedKeysSeq = "\x1b[>4;0m"
)

// newLineKeys insert a line break instead of submitting. "alt+enter" is what
// shift+enter is rewritten to; ctrl+j stays as a fallback for terminals that
// send shift+enter as a bare carriage return.
var newLineKeys = []string{"alt+enter", "ctrl+j"}

// terminalInput is shared by every program in the app so that a sequence split
// across two reads is not lost when one program hands over to the next.
var terminalInput = NewShiftEnterInput()

func requestExtendedKeys() {
	os.Stdout.WriteString(requestExtendedKeysSeq)
}

func releaseExtendedKeys() {
	os.Stdout.WriteString(releaseExtendedKeysSeq)
}

var (
	csiPrefix = []byte{0x1b, '['}
	altEnter  = []byte{0x1b, '\r'}
)

const (
	enterKeyCode  = 13 // the CR key code shift+enter reports itself as
	shiftModifier = 1  // bit 0 of the modifier mask, which is sent 1-based
)

// ShiftEnterInput wraps the terminal's stdin and rewrites shift+enter into
// alt+enter on the way to bubbletea. It stays a term.File so bubbletea still
// puts the terminal into raw mode and can cancel a pending read.
type ShiftEnterInput struct {
	tty     *os.File
	buffer  []byte // read from the terminal, not yet translated
	pending []byte // translated, not yet handed to the caller
	scratch [256]byte
}

func NewShiftEnterInput() *ShiftEnterInput {
	return &ShiftEnterInput{tty: os.Stdin}
}

func (input *ShiftEnterInput) Read(p []byte) (int, error) {
	if len(input.pending) > 0 {
		return input.drain(p), nil
	}

	read, err := input.tty.Read(input.scratch[:])
	input.buffer = append(input.buffer, input.scratch[:read]...)
	input.buffer, input.pending = translateShiftEnter(input.buffer, input.pending)
	if err != nil {
		// Nothing more is coming, so a half-read sequence is just bytes now.
		input.pending, input.buffer = append(input.pending, input.buffer...), nil
		if len(input.pending) == 0 {
			return 0, err
		}
	}

	// Zero bytes here means everything read so far was the front of an
	// unfinished escape sequence; bubbletea simply waits for the rest.
	return input.drain(p), nil
}

func (input *ShiftEnterInput) drain(p []byte) int {
	copied := copy(p, input.pending)
	if copied == len(input.pending) {
		input.pending = nil
	} else {
		input.pending = input.pending[copied:]
	}
	return copied
}

func (input *ShiftEnterInput) Write(p []byte) (int, error) { return input.tty.Write(p) }
func (input *ShiftEnterInput) Fd() uintptr                 { return input.tty.Fd() }
func (input *ShiftEnterInput) Name() string                { return input.tty.Name() }

// Close is a no-op: the wrapper borrows stdin, it does not own it.
func (input *ShiftEnterInput) Close() error { return nil }

// translateShiftEnter moves everything it can from buffer to pending, rewriting
// shift+enter along the way, and returns the tail of buffer that is still an
// unfinished escape sequence.
func translateShiftEnter(buffer, pending []byte) (rest, translated []byte) {
	for len(buffer) > 0 {
		start := bytes.Index(buffer, csiPrefix)
		if start < 0 {
			return nil, append(pending, buffer...)
		}
		pending = append(pending, buffer[:start]...)
		buffer = buffer[start:]

		sequence, complete, unfinished := splitCsi(buffer)
		switch {
		case complete && isShiftEnter(sequence):
			pending = append(pending, altEnter...)
			buffer = buffer[len(sequence):]
		case complete:
			pending = append(pending, sequence...)
			buffer = buffer[len(sequence):]
		case unfinished:
			return buffer, pending
		default:
			// Not a CSI sequence at all; pass the escape through and resync.
			pending = append(pending, buffer[:len(csiPrefix)]...)
			buffer = buffer[len(csiPrefix):]
		}
	}

	return nil, pending
}

// splitCsi splits a CSI sequence off the front of buffer. complete reports
// whether the whole sequence is there; when it is not, unfinished says whether
// what is there can still grow into one, as opposed to being something else
// that should just be passed through.
func splitCsi(buffer []byte) (sequence []byte, complete, unfinished bool) {
	for ptr := len(csiPrefix); ptr < len(buffer); ptr++ {
		switch b := buffer[ptr]; {
		case b >= 0x40 && b <= 0x7e: // final byte
			return buffer[:ptr+1], true, false
		case b >= 0x30 && b <= 0x3f: // parameter byte
		default:
			return nil, false, false
		}
	}

	return nil, false, true
}

// isShiftEnter reports whether sequence is shift+enter in either encoding
// tmux's extended-keys-format can produce: `CSI 13 ; mods u` for csi-u, or
// `CSI 27 ; mods ; 13 ~` for xterm.
func isShiftEnter(sequence []byte) bool {
	params := strings.Split(string(sequence[len(csiPrefix):len(sequence)-1]), ";")
	var code, modifiers string
	switch final := sequence[len(sequence)-1]; {
	case final == 'u' && len(params) == 2:
		code, modifiers = params[0], params[1]
	case final == '~' && len(params) == 3 && params[0] == "27":
		code, modifiers = params[2], params[1]
	default:
		return false
	}

	if code != strconv.Itoa(enterKeyCode) {
		return false
	}

	// The modifier mask is sent one-based, so subtract before testing the bit.
	mask, err := strconv.Atoi(modifiers)
	return err == nil && (mask-1)&shiftModifier != 0
}
