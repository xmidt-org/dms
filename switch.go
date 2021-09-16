package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/fx"
)

const (
	// DefaultSource is the postpone source used when no source is supplied
	DefaultSource = "<unset>"

	// DefaultTTL is the time-to-live for switches when no TTL is supplied or
	// when the TTL is nonpositive.
	DefaultTTL time.Duration = 1 * time.Minute

	// DefaultMisses is the number of allowed missed postpones before triggering
	// actions when the misses are not supplied or are nonpositive.
	DefaultMisses = 0
)

var (
	// ErrSwitchStarted is returned by Switch.Start if a Switch is currently running.
	ErrSwitchStarted = errors.New("That switch has already been started")

	// ErrSwitchStopped is returned by Switch.Stop if a Switch is not running.
	ErrSwitchStopped = errors.New("That swith has not been started")
)

type newTimer func(time.Duration) (<-chan time.Time, func() bool)

func defaultNewTimer(d time.Duration) (<-chan time.Time, func() bool) {
	t := time.NewTimer(d)
	return t.C, t.Stop
}

// PostponeRequest carries information about a postponement to a Switch.
type PostponeRequest struct {
	// Source is an identifier for the entity that is postponing the actions.
	Source string

	// RemoteAddr is the remote IP address from which the postpone request came.
	// This field can be unset for requests which do not come from a network connection.
	RemoteAddr string
}

// String returns a human-readable representation of this request.  This is the string
// output to stdout each time a postpone request is received.
func (pr PostponeRequest) String() string {
	source := pr.Source
	if len(source) == 0 {
		source = DefaultSource
	}

	if len(pr.RemoteAddr) > 0 {
		return fmt.Sprintf("[source=%s] [remoteaddr=%s]", source, pr.RemoteAddr)
	} else {
		return fmt.Sprintf("[source=%s]", source)
	}
}

// Postponer represents something that can postpone triggering actions.
type Postponer interface {
	Postpone(PostponeRequest) bool
}

// Switch is a dead man's switch.  This type is associated with a slice of Actions which
// will be executed unless postponed within a certain time-to-live interval.
type Switch struct {
	logger Logger

	ttl       time.Duration
	maxMisses int
	actions   []Action

	newTimer newTimer

	stateLock sync.Mutex
	postpone  chan PostponeRequest
	cancel    chan struct{}
}

// NewSwitch constructs a Switch.  If actions is empty, the returned Switch won't do
// anything when triggered.
func NewSwitch(l Logger, ttl time.Duration, maxMisses int, actions ...Action) *Switch {
	if ttl <= 0 {
		ttl = DefaultTTL
	}

	if maxMisses < 1 {
		maxMisses = DefaultMisses
	}

	return &Switch{
		logger:    l,
		ttl:       ttl,
		maxMisses: maxMisses,
		actions:   actions,
		newTimer:  defaultNewTimer,
	}
}

// cleanup handles shutdown of this switch's concurrent resources.  If the switch
// was actually stopped, this method returns true.  If the switch wasn't running,
// this method returns false.
func (s *Switch) cleanup() (stopped bool) {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()

	stopped = s.cancel != nil
	if stopped {
		close(s.cancel)
	}

	s.postpone = nil
	s.cancel = nil
	return
}

func (s *Switch) loop(postpone <-chan PostponeRequest, cancel <-chan struct{}) {
	defer s.cleanup()
	var misses int

	for {
		timer, stop := s.newTimer(s.ttl)
		select {
		case pr := <-postpone:
			stop()
			misses = 0

			s.logger.Printf("postponed %s", pr)

		case <-cancel:
			stop()
			s.logger.Printf("stopping switch loop")
			return

		case <-timer:
			misses++
			s.logger.Printf("missed postpone update [misses=%d]", misses)

			if misses >= s.maxMisses {
				s.logger.Printf("triggering actions")
				for _, a := range s.actions {
					s.logger.Printf("[%s]", a.String())
					if err := a.Run(); err != nil {
						s.logger.Printf("action error: %s", err)
					}
				}

				return
			}
		}
	}
}

func (s *Switch) Postpone(u PostponeRequest) (postponed bool) {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()

	if s.postpone != nil {
		postponed = true
		s.postpone <- u
	}

	return
}

// Start begins waiting for postpone requests.  If no request is received in the time-to-live
// interval, the actions will trigger.
//
// If the actions trigger, or if Stop is called, this method may be called again.  If this
// switch has already been started but has not triggered its actions yet, ErrSwitchStarted is returned.
func (s *Switch) Start(context.Context) error {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()

	if s.cancel != nil {
		return ErrSwitchStarted
	}

	s.postpone = make(chan PostponeRequest, 1)
	s.cancel = make(chan struct{})
	go s.loop(s.postpone, s.cancel)
	return nil
}

// Stop discontinues waiting for postpone requests.  The actions will not be triggered, unless
// they were already triggered by missed postpone requests.
//
// If this switch wasn't running or had already triggered actions, this method returns ErrSwitchStopped.
//
// If this switch was running and had not triggered, this method returns nil.
func (s *Switch) Stop(context.Context) (err error) {
	if !s.cleanup() {
		err = ErrSwitchStopped
	}

	return
}

func provideSwitch() fx.Option {
	return fx.Options(
		fx.Provide(
			func(cl CommandLine, l Logger, actions []Action) (*Switch, Postponer) {
				s := NewSwitch(l, cl.TTL, cl.Misses, actions...)
				return s, s
			},
		),
		fx.Invoke(
			func(l fx.Lifecycle, s *Switch) {
				l.Append(fx.Hook{
					OnStart: s.Start,
					OnStop:  s.Stop,
				})
			},
		),
	)
}
