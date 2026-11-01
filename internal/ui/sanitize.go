package ui

import (
	"strings"
	"unicode"
)

// MaxMessageLength is the longest message, in characters, the lobby accepts.
const MaxMessageLength = 256

// SanitizeMessage makes user text safe to print in other people's terminals.
// It drops control characters (including ESC, so no ANSI escape sequences can
// get through), turns tabs and newlines into spaces, removes bidirectional
// override characters that could visually reorder text to spoof it, trims the
// result and caps its length.
func SanitizeMessage(text string) string {
	var b strings.Builder
	n := 0
	for _, r := range text {
		if n >= MaxMessageLength {
			break
		}
		switch {
		case r == '\t' || r == '\n' || r == '\r':
			r = ' '
		case unicode.IsControl(r) || isBidiControl(r) || r == unicode.ReplacementChar:
			continue
		}
		b.WriteRune(r)
		n++
	}
	return strings.TrimSpace(b.String())
}

func isBidiControl(r rune) bool {
	switch {
	case r >= '‪' && r <= '‮', // embeddings and overrides
		r >= '⁦' && r <= '⁩',         // isolates
		r == '‎', r == '‏', r == '؜': // directional marks
		return true
	}
	return false
}
