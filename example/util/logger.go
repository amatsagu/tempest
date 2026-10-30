package util

import (
	"log"
	"os"
)

// Logger is the example's shared logger. Tempest itself logs through the
// standard library's log.Logger, so the example uses the same logger and format
// instead of a separate structured logging library.
var Logger = log.New(os.Stdout, "[EXAMPLE] ", log.LstdFlags)
