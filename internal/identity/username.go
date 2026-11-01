// Package identity decides who a connecting user is: which usernames are
// allowed and which SSH key each username belongs to.
package identity

import (
	"regexp"
	"strings"
)

// MaxUsernameLength keeps names short enough for the chat layout.
const MaxUsernameLength = 20

// validUsername allows only characters that are safe to print in any
// terminal, so a username can never smuggle escape sequences into the chat.
var validUsername = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// NormalizeUsername lowercases a username so that "Wes" and "wes" are the
// same person.
func NormalizeUsername(name string) string {
	return strings.ToLower(name)
}

// ValidUsername reports whether a normalized username can be used.
func ValidUsername(name string) bool {
	return len(name) <= MaxUsernameLength && validUsername.MatchString(name)
}
