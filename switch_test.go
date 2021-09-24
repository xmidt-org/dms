package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type SwitchSuite struct {
	suite.Suite

	logger      Logger
	actionLabel string
}

var _ suite.BeforeTest = (*SwitchSuite)(nil)

func (suite *SwitchSuite) BeforeTest(suiteName, testName string) {
	suite.logger = testLogger{
		suiteName: suiteName,
		testName:  testName,
	}

	suite.actionLabel = fmt.Sprintf("[%s][%s]", suiteName, testName)
}

func (suite *SwitchSuite) newActions(count int) (actions []Action) {
	for i := 0; i < count-1; i++ {
		actions = append(
			actions,
			testAction{
				t:     suite.T(),
				label: suite.actionLabel + "." + strconv.Itoa(i),
			},
		)
	}

	if count > 0 {
		actions = append(
			actions,
			testAction{
				t:     suite.T(),
				label: suite.actionLabel + "." + strconv.Itoa(count-1),
				err:   errors.New("expected error from last action"),
			},
		)
	}

	return
}

func (suite *SwitchSuite) TestTrigger() {
	actions := suite.newActions(2)
	s := NewSwitch(suite.logger, 10*time.Minute, 0, actions...)

	var timers <-chan *testTimer
	s.newTimer, timers = newTestTimer(suite.T())

	ctx := context.Background()
	suite.Equal(ErrSwitchStopped, s.Stop(ctx))
	assertNotCalled(actions)

	suite.NoError(s.Start(ctx))
	tt := waitForTimer(suite.T(), timers)
	tt.c <- time.Now()
}

func TestSwitch(t *testing.T) {
	suite.Run(t, new(SwitchSuite))
}
