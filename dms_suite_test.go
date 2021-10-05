package main

import (
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/chronon"
	"go.uber.org/fx"
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
		suiteName: suiteName,
		testName:  testName,
	}
}

func (suite *DMSSuite) clock() *chronon.FakeClock {
	return chronon.NewFakeClock(suite.now)
}

func (suite *DMSSuite) provideClock() fx.Option {
	return fx.Provide(
		suite.clock,
		func(fc *chronon.FakeClock) chronon.Clock {
			return fc
		},
	)
}

func (suite *SwitchSuite) provideActions(count int) fx.Option {
	return fx.Provide(
		func() ([]Action, []*mockAction) {
			return NewMockActions(count)
		},
	)
}
