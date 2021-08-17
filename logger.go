package main

import (
	"fmt"
	"io"
	"os"

	"go.uber.org/fx"
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

type DiscardLogger struct{}

func (dl DiscardLogger) Printf(string, ...interface{}) {}

func provideLogger() fx.Option {
	return fx.Provide(
		func() Logger {
			return WriterLogger{Writer: os.Stdout}
		},
	)
}
