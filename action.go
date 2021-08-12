package main

import (
	"errors"
	"os"
	"os/exec"
	"strings"

	"go.uber.org/fx"
)

var (
	ErrEmptyCommand = errors.New("A non-empty command is required")
)

type Action interface {
	String() string
	Execute() error
}

type CmdAction struct {
	Cmd *exec.Cmd
}

func (ca CmdAction) String() string {
	return ca.Cmd.String()
}

func (ca CmdAction) Execute() error {
	return ca.Cmd.Run()
}

func ParseExec(cl *CommandLine) ([]Action, error) {
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
		actions = append(actions,
			CmdAction{Cmd: cmd},
		)
	}

	return actions, nil
}

type ShutdownerAction struct {
	Shutdowner fx.Shutdowner
}

func (sa ShutdownerAction) String() string {
	return "Shutdown"
}

func (sa ShutdownerAction) Execute() error {
	return sa.Shutdowner.Shutdown()
}
