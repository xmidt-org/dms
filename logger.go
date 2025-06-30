// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"

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

// logServerError logs an error if and only if err is not http.ErrServerClosed.
// This function is intended to report any unexpected error from http.Error.Serve.
func logServerError(l Logger, err error) {
	if !errors.Is(err, http.ErrServerClosed) {
		l.Printf("HTTP server error: %s", err)
	}
}
