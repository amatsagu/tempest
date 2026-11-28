package util

import (
	"log"
	"os"
)

var Logger = log.New(os.Stdout, "[EXAMPLE] ", log.LstdFlags)
