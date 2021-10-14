package main

import (
	"fmt"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/fx"
)

type mockPostponer struct {
	mock.Mock
}

var _ Postponer = (*mockPostponer)(nil)

func (m *mockPostponer) Postpone(r PostponeRequest) bool {
	args := m.Called(r)
	return args.Bool(0)
}

func (m *mockPostponer) ExpectPostpone(request interface{}) *mock.Call {
	return m.On("Postpone", request)
}

type mockAction struct {
	mock.Mock
	label string
}

// String is not mocked, so that we can provide good debugging in tests
func (m *mockAction) String() string {
	return m.label
}

func (m *mockAction) Run() error {
	return m.Called().Error(0)
}

func (m *mockAction) ExpectRun() *mock.Call {
	return m.On("Run")
}

// mockActions is a slice of mocked Action instances with some useful behavior
type mockActions []*mockAction

// actions returns a new, distinct slice whose element type is Action.  That allows
// these mocks to be passed to Switches.
func (ma mockActions) actions() []Action {
	actions := make([]Action, 0, len(ma))
	for _, e := range ma {
		actions = append(actions, e)
	}

	return actions
}

// expectRunOnce sets an expectation for all actions for Run to be invoked
// exactly once and return the given error.  The returned channel will be
// signaled once for each time a mock's Run is called.
func (ma mockActions) expectRunOnce(expectedErr error) <-chan int {
	calls := make(chan int, len(ma))
	for i, e := range ma {
		i := i
		e.ExpectRun().Run(func(mock.Arguments) {
			calls <- i
		}).Return(expectedErr).Once()
	}

	return calls
}

func (ma mockActions) waitForCalls(t assert.TestingT, waitFor time.Duration, calls <-chan int) bool {
	if len(ma) > 0 {
		timer := time.NewTimer(waitFor)
		defer timer.Stop()

		for i := 0; i < len(ma); i++ {
			select {
			case <-calls:
				// passing

			case <-timer.C:
				assert.Failf(t, "Mocked actions were not invoked", "waitFor: %s", waitFor)
				return false
			}
		}
	}

	return true
}

// assertExpectations asserts all mock action expectations.
func (ma mockActions) assertExpectations(t mock.TestingT) {
	for _, e := range ma {
		e.AssertExpectations(t)
	}
}

func newMockActions(count int) mockActions {
	ma := make(mockActions, 0, count)
	for i := 0; i < count; i++ {
		ma = append(ma, &mockAction{
			label: fmt.Sprintf("action[%d]", i),
		})
	}

	return ma
}

type mockShutdowner struct {
	mock.Mock
}

func (m *mockShutdowner) Shutdown(o ...fx.ShutdownOption) error {
	args := m.Called(o)
	return args.Error(0)
}
