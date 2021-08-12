package main

import (
	"fmt"
	"io"
)

type Logger interface {
	Printf(string, ...interface{})
}

type WriterLogger struct {
	Writer io.Writer
}

func (wl WriterLogger) Printf(format string, args ...interface{}) {
	_, err := fmt.Fprintf(wl.Writer, format+"\n", args...)
	if err != nil {
		panic(err)
	}
}
