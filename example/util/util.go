// Package util holds the small helpers shared by the example features.
package util

// Ptr returns a pointer to v, for optional pointer fields on Discord objects.
func Ptr[T any](v T) *T { return &v }
