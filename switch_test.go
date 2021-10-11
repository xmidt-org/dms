package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/chronon"
)

type SwitchSuite struct {
	DMSSuite
}

func (suite *SwitchSuite) testCancelBeforeTrigger(actionCount int) {
	suite.Run("NewSwitch", func() {
		var (
			actions, mockActions = NewMockActions(actionCount)
			cfg, clock           = suite.switchConfig(0, 0, actions...)
			s                    = suite.newSwitch(cfg)
			done                 = make(chan error)
			onTicker             = make(chan chronon.FakeTicker, 1)
		)

		clock.NotifyOnTicker(onTicker)
		go func() {
			done <- s.Activate()
		}()

		<-onTicker
		suite.Equal(ErrActive, s.Activate()) // idempotent
		suite.NoError(s.Deactivate())
		suite.Equal(ErrDeactivated, <-done)
		suite.assertActionExpectations(mockActions...)
	})

	suite.Run("provideSwitch", func() {
		var (
			actions, mockActions = NewMockActions(actionCount)
			cfg, clock           = suite.switchConfig(0, 0, actions...)
			s                    *Switch
			p                    Postponer
			app                  = suite.provideSwitch(cfg, &s, &p)
			onTicker             = make(chan chronon.FakeTicker, 1)
		)

		suite.Require().NotNil(s)
		suite.Require().NotNil(p)
		clock.NotifyOnTicker(onTicker)
		app.RequireStart()

		<-onTicker
		suite.Equal(ErrActive, s.Activate()) // idempotent

		app.RequireStop()
		suite.assertActionExpectations(mockActions...)
	})
}

func (suite *SwitchSuite) TestCancelBeforeTrigger() {
	for _, actionCount := range []int{0, 1, 2, 5} {
		suite.Run(fmt.Sprintf("actionCount=%d", actionCount), func() {
			suite.testCancelBeforeTrigger(actionCount)
		})
	}
}

func TestSwitch(t *testing.T) {
	suite.Run(t, new(SwitchSuite))
}
