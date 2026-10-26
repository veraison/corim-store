package util

import (
	"iter"
	"slices"
	"strings"
)

// Normalize returns a normalized version of a name.
func Normalize(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "-", "_")
	return name
}

// IsSignedCoRIM returns true if the provided buffer appears to contain a
// signed CoRIM (no validation is performed).
func IsSignedCoRIM(buf []byte) bool {
	if len(buf) == 0 {
		return false
	}

	// tag 18 -> COSE_Sign1 -> signed corim
	return buf[0] == 0xd2
}

// IsUnignedCoRIM returns true if the provided buffer appears to contain an
// unsigned CoRIM (no validation is performed).
func IsUnsignedCoRIM(buf []byte) bool {
	if len(buf) < 3 {
		return false
	}

	// tag 501 -> unsigned corim
	return slices.Equal(buf[:3], []byte{0xd9, 0x01, 0xf5})
}

// Enumerate transforms an iter.Seq[T] into iter.Seq2[int, T], with the
// first value in Seq being the ordinal (starting at 0) of the associated
// T within the sequence.
// (side note: this really ought to be part of iter package...)
func Enumerate[T any](seq iter.Seq[T]) iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		i := 0
		for v := range seq {
			if !yield(i, v) {
				return
			}
			i++
		}
	}
}
