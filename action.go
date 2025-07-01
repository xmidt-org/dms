// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"os"
	"os/exec"
	"strings"

	"go.uber.org/fx"
)

var (
	// ErrEmptyCommand is returned by ParseExec to indicate that an exec action
	// was blank or had a blank command path.
	ErrEmptyCommand = errors.New("A non-empty command is required")
)

// Action represents something that will trigger unless postponed.
// The type *os/exec.Cmd implements this interface.
type Action interface {
	String() string
	Run() error
}

// Trigger executes each action in sequence, providing a standard output
// format for each action.
func Trigger(l Logger, actions ...Action) {
	for _, a := range actions {
		l.Printf("[%s]", a.String())
		if err := a.Run(); err != nil {
			l.Printf("action error: %s", err)
		}
	}
}

// ParseExec parses the executable actions from a command line.
func ParseExec(cl CommandLine) ([]Action, error) {
	actions := make([]Action, 0, len(cl.Exec))

	for _, e := range cl.Exec {
		pieces := strings.Split(e, " ")
		if len(pieces) == 0 || len(pieces[0]) == 0 {
			return nil, ErrEmptyCommand
		}

		cmd := exec.Command(pieces[0], pieces[1:]...)
		cmd.Dir = cl.Dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		actions = append(actions, cmd)
	}

	return actions, nil
}

// ShutdownerAction allows an uber/fx.Shutdowner to be used as an Action.
// This type is used to ensure that after trigger actions, the process exits.
type ShutdownerAction struct {
	Shutdowner fx.Shutdowner
}

func (sa ShutdownerAction) String() string {
	return "Shutdowner"
}

func (sa ShutdownerAction) Run() error {
	return sa.Shutdowner.Shutdown()
}

func provideActions() fx.Option {
	return fx.Provide(
		func(cl CommandLine, s fx.Shutdowner) (actions []Action, err error) {
			actions, err = ParseExec(cl)
			if err == nil {
				actions = append(actions, ShutdownerAction{Shutdowner: s})
			}

			return
		},
	)
}
