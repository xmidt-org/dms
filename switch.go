package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/xmidt-org/chronon"
	"go.uber.org/fx"
)

const (
	// DefaultSource is the postpone source used when no source is supplied
	DefaultSource = "<unset>"

	// DefaultTTL is the time-to-live for switches when no TTL is supplied or
	// when the TTL is nonpositive.
	DefaultTTL time.Duration = 1 * time.Minute

	// DefaultMaxMisses is the number of allowed missed postpones before triggering
	// actions when the misses are not supplied or are nonpositive.
	DefaultMaxMisses = 0
)

var (
	// ErrSwitchStarted is returned by Switch.Start if a Switch is currently running.
	ErrSwitchStarted = errors.New("That switch has already been started")

	// ErrSwitchStopped is returned by Switch.Stop if a Switch is not running.
	ErrSwitchStopped = errors.New("That swith has not been started")
)

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
	// Postpone issues a request that the action trigger be delayed by
	// at least the TTL amount.  This method returns true if the actions
	// were postponed, false if the actions had already been triggered.
	Postpone(PostponeRequest) bool
}

// SwitchConfig represents the set of configurable options for a Switch.
type SwitchConfig struct {
	// Logger is the required sink for logging output.
	Logger Logger

	// TTL is the interval on which the switch sleeps, waiting for postpones.
	// When this interval elapses MaxMisses number of times with no postpones,
	// the switch triggers its actions.
	TTL time.Duration

	// MaxMisses is the number of missed postpones that are allowed before
	// actions trigger.
	MaxMisses int

	// Actions are the set of tasks to trigger when the Switch's interval
	// elapses without being postponed.
	Actions []Action

	// Clock is the required source of time information.
	Clock chronon.Clock
}

// Switch is a dead man's switch.  This type is associated with a slice of Actions which
// will be executed unless postponed within a certain time-to-live interval.
type Switch struct {
	logger Logger

	ttl       time.Duration
	maxMisses int
	actions   []Action

	clock chronon.Clock

	stateLock sync.Mutex
	postpone  chan PostponeRequest
	cancel    chan struct{}
}

// NewSwitch constructs a Switch using the given set of configuration options.
func NewSwitch(cfg SwitchConfig) *Switch {
	s := &Switch{
		logger:    cfg.Logger,
		ttl:       cfg.TTL,
		maxMisses: cfg.MaxMisses,
		actions:   cfg.Actions,
		clock:     cfg.Clock,
	}

	if s.ttl <= 0 {
		s.ttl = DefaultTTL
	}

	if s.maxMisses <= 0 {
		s.maxMisses = DefaultMaxMisses
	}

	return s
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
	t := s.clock.NewTicker(s.ttl)
	defer t.Stop()

	for {
		select {
		case pr := <-postpone:
			t.Reset(s.ttl)
			misses = 0

			s.logger.Printf("postponed %s", pr)

		case <-cancel:
			s.logger.Printf("stopping switch loop")
			return

		case <-t.C():
			misses++
			s.logger.Printf("missed postpone update [misses=%d]", misses)

			if misses >= s.maxMisses {
				s.logger.Printf("triggering actions")
				Trigger(s.logger, s.actions...)
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

// SwitchIn holds the set of dependencies required to create a Switch.
type SwitchIn struct {
	fx.In

	Logger  Logger
	Actions []Action
	Config  SwitchConfig
	Clock   chronon.Clock
}

func provideSwitch() fx.Option {
	return fx.Options(
		fx.Provide(
			func(l Logger, cl CommandLine, clock chronon.Clock, actions []Action) SwitchConfig {
				return SwitchConfig{
					Logger:    l,
					Actions:   actions,
					TTL:       cl.TTL,
					MaxMisses: cl.Misses,
					Clock:     clock,
				}
			},
			func(in SwitchIn) (*Switch, Postponer) {
				s := NewSwitch(in.Config)
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
