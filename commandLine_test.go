package main

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type CommandLineSuite struct {
	DMSSuite
}

func (suite *CommandLineSuite) TestError() {
	app := fx.New(
		fx.Logger(DiscardLogger{}),
		parseCommandLine([]string{"--unrecognized"}),
	)

	suite.Error(app.Err())
}

func (suite *CommandLineSuite) TestTypical() {
	app := fxtest.New(
		suite.T(),
		fx.Logger(DiscardLogger{}),
		parseCommandLine([]string{
			"--http", ":8080",
			"--ttl", "10s",
			"--exec", "echo 'hi there'",
		}),
	)

	app.RequireStart()
	app.RequireStop()
}

func (suite *CommandLineSuite) TestDebug() {
	app := fxtest.New(
		suite.T(),
		fx.Logger(DiscardLogger{}),
		parseCommandLine([]string{
			"--http", ":8080",
			"--ttl", "10s",
			"--exec", "echo 'hi there'",
			"--debug", // NOTE: this will cause output from uber/fx in test output
		}),
	)

	app.RequireStart()
	app.RequireStop()
}

func TestCommandLine(t *testing.T) {
	suite.Run(t, new(CommandLineSuite))
}
