//go:build test

package util

// Ptr returns the pointer to the specified value. This is useful when specifying
// literal values for fields that pointers to types for which directly taking a
// pointer of a literal is not possible (e.g. `Field: &"foo"` is not valid, do
// `Field: Ptr("foo")` instead).
func Ptr[T any](val T) *T {
	return &val
}
