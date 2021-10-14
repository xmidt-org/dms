package main

import (
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/chronon"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

// DMSSuite hosts common infrastructure for dms unit test suites.
type DMSSuite struct {
	suite.Suite

	now    time.Time
	logger Logger
}

var _ suite.BeforeTest = (*DMSSuite)(nil)
var _ suite.SetupAllSuite = (*DMSSuite)(nil)

func (suite *DMSSuite) SetupSuite() {
	suite.now = time.Now()
}

func (suite *DMSSuite) BeforeTest(suiteName, testName string) {
	suite.logger = testLogger{
		t:         suite.T(),
		suiteName: suiteName,
		testName:  testName,
	}
}

func (suite *DMSSuite) provideLogger() fx.Option {
	return fx.Provide(
		func() Logger {
			return suite.logger
		},
	)
}

func (suite *DMSSuite) clock() *chronon.FakeClock {
	return chronon.NewFakeClock(suite.now)
}

// switchConfig returns a SwitchConfig built with this suite's current state.
// A fake clock is used to control any ticker loops started by tests.
func (suite *DMSSuite) switchConfig(ttl time.Duration, maxMisses int, actions ...Action) (SwitchConfig, *chronon.FakeClock) {
	clock := suite.clock()
	cfg := SwitchConfig{
		Logger:    suite.logger,
		TTL:       ttl,
		MaxMisses: maxMisses,
		Actions:   actions,
		Clock:     clock,
	}

	return cfg, clock
}

// newSwitch uses NewSwitch to directly construct a Switch.
func (suite *DMSSuite) newSwitch(cfg SwitchConfig) *Switch {
	s := NewSwitch(cfg)
	suite.Require().NotNil(s)
	return s
}

// provideSwitch uses an enclosing fx.App to create a Switch.
func (suite *DMSSuite) provideSwitch(cfg SwitchConfig, populate ...interface{}) *fxtest.App {
	app := fxtest.New(
		suite.T(),
		fx.Supply(cfg),
		provideSwitch(),
		fx.Populate(populate...),
	)

	suite.Require().NoError(app.Err())
	return app
}
