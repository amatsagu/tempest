package other

import (
	"log"
	"runtime"
)

func FormatError(err error) {
	if err == nil {
		return
	}

	pc, fn, line, _ := runtime.Caller(1)
	log.Printf("error at %s[%s:%d] :: %v\n", runtime.FuncForPC(pc).Name(), fn, line, err)
}
