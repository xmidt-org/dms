package main

import (
	"github.com/stretchr/testify/mock"
	"go.uber.org/fx"
)

type mockShutdowner struct {
	mock.Mock
}

func (m *mockShutdowner) Shutdown(o ...fx.ShutdownOption) error {
	args := m.Called(o)
	return args.Error(0)
}
