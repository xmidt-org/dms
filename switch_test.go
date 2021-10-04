package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/chronon"
	"go.uber.org/fx"
)

type SwitchSuite struct {
	suite.Suite

	now    time.Time
	logger Logger
}

var _ suite.BeforeTest = (*SwitchSuite)(nil)
var _ suite.SetupAllSuite = (*SwitchSuite)(nil)

func (suite *SwitchSuite) SetupSuite() {
	suite.now = time.Now()
}

func (suite *SwitchSuite) BeforeTest(suiteName, testName string) {
	suite.logger = testLogger{
		suiteName: suiteName,
		testName:  testName,
	}
}

// actions emits the given number of mocked actions into the enclosing application.
// Both an []Action and an []*mockAction are emitted.
func (suite *SwitchSuite) actions(count int) fx.Option {
	return fx.Provide(
		func() ([]Action, []*mockAction) {
			return NewMockActions(count)
		},
	)
}

func (suite *SwitchSuite) clock() fx.Option {
	return fx.Provide(
		func() (chronon.Clock, *chronon.FakeClock) {
			fc := chronon.NewFakeClock(suite.now)
			return fc, fc
		},
	)
}

func (suite *SwitchSuite) TestTrigger() {
	testCases := []CommandLine{
		CommandLine{}, // defaults
		CommandLine{
			TTL:    50 * time.Minute,
			Misses: 2,
		},
	}
}

func TestSwitch(t *testing.T) {
	suite.Run(t, new(SwitchSuite))
}
