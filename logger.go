package main

import (
	"bytes"
	"fmt"
	"io"

	"go.uber.org/fx"
)

// Logger is the logging interface expected by dms.  Loggers in
// various packages, such as go.uber.org/fx, implement this interface.
type Logger interface {
	Printf(string, ...interface{})
}

// WriterLogger is a Logger that sends its output to a given io.Writer.
type WriterLogger struct {
	Writer io.Writer
}

func (wl WriterLogger) Printf(format string, args ...interface{}) {
	var o bytes.Buffer
	fmt.Fprintf(&o, format+"\n", args...)

	// ensure a single write for each printf
	_, err := wl.Writer.Write(o.Bytes())
	if err != nil {
		panic(err)
	}
}

// DiscardLogger is a Logger that ignores all output.
type DiscardLogger struct{}

func (dl DiscardLogger) Printf(string, ...interface{}) {}

func provideLogger(w io.Writer) fx.Option {
	return fx.Provide(
		func() Logger {
			return WriterLogger{Writer: w}
		},
	)
}
