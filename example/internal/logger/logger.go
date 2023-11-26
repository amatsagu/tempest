package logger

import (
	"log"
	"os"
)

var (
	Info  *log.Logger
	Warn  *log.Logger
	Error *log.Logger
)

func InitLogger() {
	Info = log.New(os.Stdout, "INFO: ", log.LstdFlags|log.Lshortfile)
	Warn = log.New(os.Stdout, "WARN: ", log.LstdFlags|log.Lshortfile)
	Error = log.New(os.Stdout, "ERROR: ", log.LstdFlags|log.Lshortfile)
}
