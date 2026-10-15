// Package util holds the small helpers shared by the example features.
package util

// Ptr returns a pointer to v, for the optional pointer fields on Discord
// objects such as a command option's maximum length.
func Ptr[T any](v T) *T { return &v }
