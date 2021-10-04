package main

import (
	"fmt"
	"testing"

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

// NewMockActions creates a slice of mock actions.  The two slices contain
// the same actions, but the first slice can be used with code under test.
func NewMockActions(count int) (actions []Action, mocks []*mockAction) {
	for i := 0; i < count; i++ {
		ma := &mockAction{
			label: fmt.Sprintf("action[%d]", i),
		}

		actions = append(actions, ma)
		mocks = append(mocks, ma)
	}

	return
}

// AssertActionExpectations is a helper for verifying zero or more mocked actions.
func AssertActionExpectations(t *testing.T, actions ...*mockAction) {
	for _, a := range actions {
		a.AssertExpectations(t)
	}
}

// ExpectRunOnce sets each action to Run exactly once.  All Run invocations
// return the given error value.
func ExpectRunOnce(err error, actions ...*mockAction) {
	for _, a := range actions {
		a.ExpectRun().Return(err).Once()
	}
}

type mockShutdowner struct {
	mock.Mock
}

func (m *mockShutdowner) Shutdown(o ...fx.ShutdownOption) error {
	args := m.Called(o)
	return args.Error(0)
}
