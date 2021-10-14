package main

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/chronon"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type SwitchConfigSuite struct {
	DMSSuite
}

func (suite *SwitchConfigSuite) TestProvideSwitchConfig() {
	suite.Run("Minimal", func() {
		var (
			mockActions = newMockActions(1)
			actions     = mockActions.actions()
			cfg         SwitchConfig

			app = fxtest.New(
				suite.T(),
				fx.Supply(actions),
				suite.provideLogger(),
				provideSwitchConfig(),
				fx.Populate(&cfg),
			)
		)

		app.RequireStart()

		suite.Equal(
			SwitchConfig{
				Logger:  suite.logger,
				Actions: actions,
			},
			cfg,
		)

		app.RequireStop()
		mockActions.assertExpectations(suite.T())
	})

	suite.Run("Full", func() {
		var (
			mockActions = newMockActions(3)
			actions     = mockActions.actions()
			clock       = suite.clock()
			cfg         SwitchConfig

			app = fxtest.New(
				suite.T(),
				fx.Supply(
					actions,
					CommandLine{
						TTL:    12 * time.Minute,
						Misses: 7,
					},
				),
				fx.Provide(
					func() chronon.Clock {
						return clock
					},
				),
				suite.provideLogger(),
				provideSwitchConfig(),
				fx.Populate(&cfg),
			)
		)

		app.RequireStart()

		suite.Equal(
			SwitchConfig{
				Logger:    suite.logger,
				Actions:   actions,
				TTL:       12 * time.Minute,
				MaxMisses: 7,
				Clock:     clock,
			},
			cfg,
		)

		app.RequireStop()
		mockActions.assertExpectations(suite.T())
	})
}

func TestSwitchConfig(t *testing.T) {
	suite.Run(t, new(SwitchConfigSuite))
}

type SwitchSuite struct {
	DMSSuite
}

func (suite *SwitchSuite) TestDefaults() {
	suite.Run("NewSwitch", func() {
		var (
			mockActions = newMockActions(1)
			actions     = mockActions.actions()

			s = suite.newSwitch(
				SwitchConfig{
					Logger:  suite.logger,
					Actions: actions,
				},
			)
		)

		suite.Equal(suite.logger, s.logger)
		suite.Equal(DefaultTTL, s.ttl)
		suite.Equal(DefaultMaxMisses, s.maxMisses)
		suite.True(chronon.IsSystemClock(s.clock))
	})

	suite.Run("provideSwitch", func() {
		var (
			mockActions = newMockActions(1)
			actions     = mockActions.actions()

			s   *Switch
			app = fxtest.New(
				suite.T(),
				suite.provideLogger(),
				fx.Supply(actions),
				provideSwitchConfig(),
				provideSwitch(),
				fx.Populate(&s),
			)
		)

		// just in case the test takes a while, stub the action
		mockActions[0].ExpectRun().Maybe().Return(error(nil))

		app.RequireStart()
		suite.Require().NotNil(s)

		suite.Equal(suite.logger, s.logger)
		suite.Equal(DefaultTTL, s.ttl)
		suite.Equal(DefaultMaxMisses, s.maxMisses)
		suite.True(chronon.IsSystemClock(s.clock))

		app.RequireStop()
	})
}

func (suite *SwitchSuite) testCancelBeforeTrigger(actionCount int) {
	suite.Run("NewSwitch", func() {
		var (
			mockActions = newMockActions(actionCount)
			cfg, clock  = suite.switchConfig(0, 0, mockActions.actions()...)
			s           = suite.newSwitch(cfg)
			done        = make(chan error)
			onTicker    = make(chan chronon.FakeTicker, 1)
		)

		clock.NotifyOnTicker(onTicker)
		go func() {
			done <- s.Activate()
		}()

		<-onTicker
		suite.Equal(ErrActive, s.Activate()) // idempotent
		suite.NoError(s.Deactivate())
		suite.Equal(ErrDeactivated, <-done)
		mockActions.assertExpectations(suite.T())
	})

	suite.Run("provideSwitch", func() {
		var (
			mockActions = newMockActions(actionCount)
			cfg, clock  = suite.switchConfig(0, 0, mockActions.actions()...)
			s           *Switch
			p           Postponer
			app         = suite.provideSwitch(cfg, &s, &p)
			onTicker    = make(chan chronon.FakeTicker, 1)
		)

		suite.Require().NotNil(s)
		suite.Require().NotNil(p)
		clock.NotifyOnTicker(onTicker)
		app.RequireStart()

		<-onTicker
		suite.Equal(ErrActive, s.Activate()) // idempotent

		app.RequireStop()
		mockActions.assertExpectations(suite.T())

		// postpone should be idempotent
		suite.False(s.Postpone(PostponeRequest{}))
	})
}

func (suite *SwitchSuite) TestCancelBeforeTrigger() {
	for _, actionCount := range []int{0, 1, 2, 5} {
		suite.Run(fmt.Sprintf("actionCount=%d", actionCount), func() {
			suite.testCancelBeforeTrigger(actionCount)
		})
	}
}

func (suite *SwitchSuite) testTrigger(actionCount int) {
	suite.Run("NewSwitch", func() {
		var (
			mockActions = newMockActions(actionCount)
			cfg, clock  = suite.switchConfig(0, 0, mockActions.actions()...)
			s           = suite.newSwitch(cfg)
			done        = make(chan error)
			onTicker    = make(chan chronon.FakeTicker, 1)
		)

		clock.NotifyOnTicker(onTicker)
		go func() {
			done <- s.Activate()
		}()

		calls := mockActions.expectRunOnce(errors.New("expected"))
		ft := <-onTicker
		suite.Equal(ErrActive, s.Activate()) // idempotent

		clock.Set(ft.When())
		mockActions.waitForCalls(suite.T(), time.Second, calls)

		suite.NoError(<-done)
		mockActions.assertExpectations(suite.T())
	})

	suite.Run("provideSwitch", func() {
		var (
			mockActions = newMockActions(actionCount)
			cfg, clock  = suite.switchConfig(0, 0, mockActions.actions()...)
			s           *Switch
			p           Postponer
			app         = suite.provideSwitch(cfg, &s, &p)
			onTicker    = make(chan chronon.FakeTicker, 1)
		)

		suite.Require().NotNil(s)
		suite.Require().NotNil(p)
		clock.NotifyOnTicker(onTicker)
		app.RequireStart()

		calls := mockActions.expectRunOnce(errors.New("expected"))
		ft := <-onTicker
		suite.Equal(ErrActive, s.Activate()) // idempotent

		clock.Set(ft.When())
		mockActions.waitForCalls(suite.T(), time.Second, calls)

		app.RequireStop()
		mockActions.assertExpectations(suite.T())

		// postpone should be idempotent
		suite.False(s.Postpone(PostponeRequest{}))
	})
}

func (suite *SwitchSuite) TestTrigger() {
	for _, actionCount := range []int{0, 1, 2, 5} {
		suite.Run(fmt.Sprintf("actionCount=%d", actionCount), func() {
			suite.testTrigger(actionCount)
		})
	}
}

func (suite *SwitchSuite) testPostpone(ttl time.Duration, actionCount, maxMisses int) {
	suite.Run("NewSwitch", func() {
		var (
			mockActions = newMockActions(actionCount)
			cfg, clock  = suite.switchConfig(ttl, maxMisses, mockActions.actions()...)
			s           = suite.newSwitch(cfg)
			done        = make(chan error)
			onTicker    = make(chan chronon.FakeTicker, 1)
		)

		clock.NotifyOnTicker(onTicker)
		go func() {
			done <- s.Activate()
		}()

		ft := <-onTicker
		suite.Equal(ErrActive, s.Activate()) // idempotent

		clock.Add(ttl / 2) // no trigger should happen yet
		suite.True(s.Postpone(PostponeRequest{Source: "test"}))

		// the ticker should be reset to suite.now plus ttl*1.5, since
		// we advanced by half the TTL first
		suite.Require().Eventually(
			func() bool { return ft.When().Equal(suite.now.Add(ttl * 3 / 2)) },
			time.Second,
			time.Second/4,
			"The ticker was not reset",
		)

		calls := mockActions.expectRunOnce(errors.New("expected"))

		// advance by steps to force misses and finally a trigger
		for i := 0; i <= maxMisses; i++ {
			clock.Add(ttl)
		}

		mockActions.waitForCalls(suite.T(), time.Second, calls)
		suite.NoError(<-done)
		mockActions.assertExpectations(suite.T())
	})

	suite.Run("provideSwitch", func() {
		var (
			mockActions = newMockActions(actionCount)
			cfg, clock  = suite.switchConfig(ttl, maxMisses, mockActions.actions()...)
			s           *Switch
			p           Postponer
			app         = suite.provideSwitch(cfg, &s, &p)
			onTicker    = make(chan chronon.FakeTicker, 1)
		)

		suite.Require().NotNil(s)
		suite.Require().NotNil(p)
		clock.NotifyOnTicker(onTicker)
		app.RequireStart()

		ft := <-onTicker
		suite.Equal(ErrActive, s.Activate()) // idempotent

		clock.Add(ttl / 2) // no trigger should happen yet
		suite.True(s.Postpone(PostponeRequest{Source: "test"}))

		// the ticker should be reset to suite.now plus ttl*1.5, since
		// we advanced by half the TTL first
		suite.Require().Eventually(
			func() bool { return ft.When().Equal(suite.now.Add(ttl * 3 / 2)) },
			time.Second,
			time.Second/4,
			"The ticker was not reset",
		)

		calls := mockActions.expectRunOnce(errors.New("expected"))

		// advance by steps to force misses and finally a trigger
		for i := 0; i <= maxMisses; i++ {
			clock.Add(ttl)
		}

		mockActions.waitForCalls(suite.T(), time.Second, calls)
		app.RequireStop()

		mockActions.assertExpectations(suite.T())
	})
}

func (suite *SwitchSuite) TestPostpone() {
	for _, actionCount := range []int{0, 1, 2, 5} {
		suite.Run(fmt.Sprintf("actionCount=%d", actionCount), func() {
			for _, maxMisses := range []int{0, 1, 2} {
				suite.Run(fmt.Sprintf("maxMisses=%d", maxMisses), func() {
					// just use the same TTL, as there's no behavioral difference based
					// on the TTL value right now
					suite.testPostpone(12*time.Second, actionCount, maxMisses)
				})
			}
		})
	}
}

func TestSwitch(t *testing.T) {
	suite.Run(t, new(SwitchSuite))
}
