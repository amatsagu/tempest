package route

import "errors"

var (
	errInvalidPattern = errors.New("route: pattern must start with '/' and use non-empty segments")
	errConflict       = errors.New("route: pattern conflicts with a previously registered one")
)
