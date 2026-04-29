package util

import (
	"os"
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

func TestIsXXXCoRIM(t *testing.T) {
	signed, err := os.ReadFile("../../sample/corim/signed-cca-ref-plat.cose")
	assert.NoError(t, err)
	unsigned, err := os.ReadFile("../../sample/corim/unsigned-cca-ref-plat.cbor")
	assert.NoError(t, err)

	assert.True(t, IsSignedCoRIM(signed))
	assert.False(t, IsSignedCoRIM(unsigned))
	assert.False(t, IsUnsignedCoRIM(signed))
	assert.True(t, IsUnsignedCoRIM(unsigned))
	assert.False(t, IsSignedCoRIM([]byte{}))
	assert.False(t, IsUnsignedCoRIM([]byte{}))
}
