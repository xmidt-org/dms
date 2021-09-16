package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/fx"
)

type testAction struct {
	t *testing.T

	label  string
	err    error
	called bool
}

func (ta testAction) String() string {
	return ta.label
}

func (ta testAction) Run() error {
	assert.Falsef(ta.t, ta.called, "Action %s has already been called", ta.label)
	return ta.err
}

type mockShutdowner struct {
	mock.Mock
}

func (m *mockShutdowner) Shutdown(o ...fx.ShutdownOption) error {
	args := m.Called(o)
	return args.Error(0)
}
