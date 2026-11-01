package ui

import "math/rand/v2"

// userColors are ANSI-256 colors that stay readable on dark terminal backgrounds.
var userColors = []string{
	"39", "42", "81", "118", "141", "171", "203", "205", "214", "220",
}

// randomUserColor picks a random color for a user, avoiding the ones in use
// while any are free.
func randomUserColor(inUse map[string]bool) string {
	free := make([]string, 0, len(userColors))
	for _, color := range userColors {
		if !inUse[color] {
			free = append(free, color)
		}
	}
	if len(free) == 0 {
		free = userColors
	}
	return free[rand.IntN(len(free))]
}
