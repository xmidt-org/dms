package main

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/fx"
)

type testTimer struct {
	t *testing.T
	d time.Duration
	c chan time.Time

	lock    sync.Mutex
	stopped chan struct{}
}

func (tt *testTimer) stop() bool {
	tt.lock.Lock()
	defer tt.lock.Unlock()

	if assert.NotNil(tt.t, tt.stopped, "this timer has already been stopped") {
		close(tt.stopped)
		tt.stopped = nil
	}

	return false
}

func (tt *testTimer) waitForStop() {
	select {
	case <-tt.stopped:
		// passing

	case <-time.After(time.Second):
		assert.Fail(tt.t, "this timer was not stopped")
	}
}

func newTestTimer(t *testing.T) (nt newTimer, ct <-chan *testTimer) {
	timers := make(chan *testTimer, 1)
	ct = timers
	nt = func(d time.Duration) (<-chan time.Time, func() bool) {
		tt := &testTimer{
			t:       t,
			d:       d,
			c:       make(chan time.Time, 1),
			stopped: make(chan struct{}),
		}

		timers <- tt
		return tt.c, tt.stop
	}

	return
}

func waitForTimer(t *testing.T, timers <-chan *testTimer) *testTimer {
	select {
	case tt := <-timers:
		// passing
		return tt

	case <-time.After(time.Second):
		assert.Fail(t, "No timer created")
		return nil
	}
}

type mockPostponer struct {
	mock.Mock
}

var _ Postponer = (*mockPostponer)(nil)

func (m *mockPostponer) Postpone(r PostponeRequest) bool {
	args := m.Called(r)
	return args.Bool(0)
}

type testAction struct {
	t *testing.T

	label  string
	err    error
	called bool
}

func (ta testAction) String() string {
	return ta.label
}

func (ta testAction) Run() error {
	ta.assertNotCalled()
	ta.called = true
	return ta.err
}

func (ta testAction) assertCalled() bool {
	return assert.Truef(ta.t, ta.called, "Action %s was not called", ta.label)
}

func (ta testAction) assertNotCalled() bool {
	return assert.Falsef(ta.t, ta.called, "Action %s was called", ta.label)
}

func assertCalled(actions []Action) bool {
	count := 0
	for _, a := range actions {
		if a.(testAction).assertCalled() {
			count++
		}
	}

	return count == len(actions)
}

func assertNotCalled(actions []Action) bool {
	count := 0
	for _, a := range actions {
		if a.(testAction).assertNotCalled() {
			count++
		}
	}

	return count == len(actions)
}

type mockShutdowner struct {
	mock.Mock
}

func (m *mockShutdowner) Shutdown(o ...fx.ShutdownOption) error {
	args := m.Called(o)
	return args.Error(0)
}
