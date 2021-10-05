package main

import (
	"context"
	"runtime"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type SwitchSuite struct {
	DMSSuite
}

func (suite *SwitchSuite) TestCancel() {
	suite.Run("NewSwitch", func() {
		var (
			actions, mockActions = NewMockActions(1)

			s = NewSwitch(SwitchConfig{
				Logger:  suite.logger,
				Actions: actions,
				Clock:   suite.clock(),
			})
		)

		suite.Require().NotNil(s)
		ctx := context.Background()

		suite.NoError(s.Start(ctx))
		runtime.Gosched()
		suite.Equal(ErrSwitchStarted, s.Start(ctx))

		suite.NoError(s.Stop(ctx))
		suite.Equal(ErrSwitchStopped, s.Stop(ctx))
		AssertActionExpectations(suite.T(), mockActions...)
	})

	suite.Run("provideSwitch", func() {
		var (
			actions, mockActions = NewMockActions(1)

			s   *Switch
			app = fxtest.New(
				suite.T(),
				fx.Supply(
					SwitchConfig{
						Logger:  suite.logger,
						Actions: actions,
						Clock:   suite.clock(),
					},
				),
				provideSwitch(),
				fx.Populate(&s),
			)
		)

		suite.Require().NotNil(s)

		app.RequireStart()
		runtime.Gosched()

		app.RequireStop()
		runtime.Gosched()

		AssertActionExpectations(suite.T(), mockActions...)
	})
}

func TestSwitch(t *testing.T) {
	suite.Run(t, new(SwitchSuite))
}
