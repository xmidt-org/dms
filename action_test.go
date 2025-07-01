// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type ActionSuite struct {
	suite.Suite

	shutdowner *mockShutdowner
}

var _ suite.SetupTestSuite = (*ActionSuite)(nil)

func (suite *ActionSuite) SetupTest() {
	suite.shutdowner = new(mockShutdowner)
}

func (suite *ActionSuite) assertCmd(cmd *exec.Cmd, expectedDir string, expectedPieces []string) {
	suite.True(
		strings.HasSuffix(
			cmd.Path, // LookupPath may have put the full command path
			expectedPieces[0],
		),
	)

	suite.Equal(
		expectedPieces,
		cmd.Args,
	)

	suite.Equal(expectedDir, cmd.Dir)
	suite.Equal(os.Stdout, cmd.Stdout)
	suite.Equal(os.Stderr, cmd.Stderr)
}

func (suite *ActionSuite) TestEmptyCommand() {
	testData := []CommandLine{
		{
			Exec: []string{""},
		},
		{
			Exec: []string{"ls", ""},
		},
		{
			Exec: []string{"", "ls"},
		},
	}

	suite.Run("ParseExec", func() {
		for i, testCase := range testData {
			suite.Run(strconv.Itoa(i), func() {
				actions, err := ParseExec(testCase)
				suite.Empty(actions)
				suite.Error(err)
			})
		}
	})

	suite.Run("provideActions", func() {
		for i, testCase := range testData {
			suite.Run(strconv.Itoa(i), func() {
				var actions []Action
				app := fx.New(
					fx.Logger(DiscardLogger{}),
					fx.Supply(testCase),
					provideActions(),
					fx.Populate(&actions),
				)

				suite.Error(app.Err())
			})
		}
	})
}

func (suite *ActionSuite) TestValidCommands() {
	testData := []struct {
		commandLine    CommandLine
		expectedPieces [][]string
	}{
		{
			commandLine: CommandLine{
				Exec: []string{`echo hello`},
			},
			expectedPieces: [][]string{
				{"echo", "hello"},
			},
		},
		{
			commandLine: CommandLine{
				Exec: []string{"ls -al", `echo test`},
			},
			expectedPieces: [][]string{
				{"ls", "-al"},
				{"echo", "test"},
			},
		},
		{
			commandLine: CommandLine{
				Exec: []string{"netstat -an", "ls", `echo another test`},
			},
			expectedPieces: [][]string{
				{"netstat", "-an"},
				{"ls"},
				{"echo", "another", "test"},
			},
		},
	}

	suite.Run("ParseExec", func() {
		for i, testCase := range testData {
			suite.Run(strconv.Itoa(i), func() {
				actions, err := ParseExec(testCase.commandLine)
				suite.Require().NoError(err)
				suite.Require().Equal(len(testCase.commandLine.Exec), len(actions))

				for j, a := range actions {
					suite.assertCmd(
						a.(*exec.Cmd),
						testCase.commandLine.Dir,
						testCase.expectedPieces[j],
					)
				}
			})
		}
	})

	suite.Run("provdeActions", func() {
		for i, testCase := range testData {
			suite.Run(strconv.Itoa(i), func() {
				var actions []Action
				fxtest.New(
					suite.T(),
					fx.Logger(DiscardLogger{}),
					fx.Supply(testCase.commandLine),
					provideActions(),
					fx.Populate(&actions),
				)

				// +1 for the shutdown action
				suite.Require().Equal(len(testCase.commandLine.Exec)+1, len(actions))
				if suite.IsType(ShutdownerAction{}, actions[len(actions)-1]) {
					sa := actions[len(actions)-1].(ShutdownerAction)
					suite.NotNil(sa.Shutdowner)
				}

				for j := 0; j < len(actions)-1; j++ {
					suite.assertCmd(
						actions[j].(*exec.Cmd),
						testCase.commandLine.Dir,
						testCase.expectedPieces[j],
					)
				}
			})
		}
	})
}

func (suite *ActionSuite) TestSuccess() {
	suite.shutdowner.On("Shutdown", []fx.ShutdownOption(nil)).Return(error(nil))
	sa := ShutdownerAction{
		Shutdowner: suite.shutdowner,
	}

	suite.NotEmpty(sa.String())
	suite.NoError(sa.Run())
	suite.shutdowner.AssertExpectations(suite.T())
}

func (suite *ActionSuite) TestError() {
	expectedErr := errors.New("expected")
	suite.shutdowner.On("Shutdown", []fx.ShutdownOption(nil)).Return(expectedErr)
	sa := ShutdownerAction{
		Shutdowner: suite.shutdowner,
	}

	suite.NotEmpty(sa.String())
	suite.Equal(expectedErr, sa.Run())
	suite.shutdowner.AssertExpectations(suite.T())
}

func TestAction(t *testing.T) {
	suite.Run(t, new(ActionSuite))
}

type ProvideActionsSuite struct {
	suite.Suite
}

func (suite *ProvideActionsSuite) TestSuccess() {
	app := fxtest.New(
		suite.T(),
		fx.Logger(DiscardLogger{}),
		provideActions(),
	)

	app.RequireStart()
	app.RequireStop()
}

func TestProvideActions(t *testing.T) {
	suite.Run(t, new(ProvideActionsSuite))
}
