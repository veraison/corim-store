package util

import "strings"

// Normalize returns a normalized version of a name.
func Normalize(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "-", "_")
	return name
}
