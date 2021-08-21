package main

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type LoggerSuite struct {
	suite.Suite

	capture *bytes.Buffer
}

var _ suite.SetupTestSuite = (*LoggerSuite)(nil)

func (suite *LoggerSuite) SetupTest() {
	suite.capture = new(bytes.Buffer)
}

func (suite *LoggerSuite) TestWriterLogger() {
	wl := WriterLogger{Writer: suite.capture}
	wl.Printf("test: %d", 123)
	suite.Equal("test: 123\n", suite.capture.String())
}

func (suite *LoggerSuite) TestWriterLoggerError() {
	r, w := io.Pipe()
	r.Close()
	w.Close()

	wl := WriterLogger{Writer: w}
	suite.Panics(func() {
		wl.Printf("test: %d", 123)
	})
}

func (suite *LoggerSuite) TestDiscardLogger() {
	DiscardLogger{}.Printf("test: %d", 123)
}

func (suite *LoggerSuite) TestProvideLogger() {
	var l Logger
	app := fxtest.New(
		suite.T(),
		fx.Logger(DiscardLogger{}),
		provideLogger(suite.capture),
		fx.Populate(&l),
	)

	app.RequireStart()
	app.RequireStop()

	suite.Require().NotNil(l)
	l.Printf("test: %d", 123)
	suite.Equal("test: 123\n", suite.capture.String())
}

func TestLogger(t *testing.T) {
	suite.Run(t, new(LoggerSuite))
}
