package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalize(t *testing.T) {
	testCases := []struct {
		name   string
		before string
		after  string
	}{
		{
			name:   "ident",
			before: "hello_world",
			after:  "hello_world",
		},
		{
			name:   "bars",
			before: "hello-world",
			after:  "hello_world",
		},
		{
			name:   "case",
			before: "Hello_World",
			after:  "hello_world",
		},
		{
			name:   "space",
			before: "  hello_world  ",
			after:  "hello_world",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualAfter := Normalize(tc.before)
			assert.Equal(t, tc.after, actualAfter)
		})
	}
}
