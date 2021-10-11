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
	// ErrActive is returned by Switch.Activate if a Switch is currently running.
	ErrActive = errors.New("That switch is already active")

	// ErrNotActive is returned by Switch.Cancel if a Switch is not running.
	ErrNotActive = errors.New("That switch is not active")

	// ErrDeactivated is returned by Activate if Deactivate was called before
	// actions were triggered.
	ErrDeactivated = errors.New("That switch has been deactivated")
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
	//
	// If nonpositive, DefaultTTL is used.
	TTL time.Duration

	// MaxMisses is the number of missed postpones that are allowed before
	// actions trigger.
	//
	// If nonpositive, DefaultMaxMisses is used.
	MaxMisses int

	// Actions are the set of tasks to trigger when the Switch's interval
	// elapses without being postponed.  If this is an empty slice, then
	// nothing happens when a switch is triggered.
	Actions []Action

	// Clock is the optional source of time information.  If unset,
	// the system clock is used.
	Clock chronon.Clock
}

// SwitchConfigIn describes all the dependencies necessary for creating a SwitchConfig.
type SwitchConfigIn struct {
	fx.In

	Logger      Logger
	Actions     []Action
	CommandLine CommandLine   `optional:"true"`
	Clock       chronon.Clock `optional:"true"`
}

// provideSwitchConfig creates a SwitchConfig from injected components.
// In particular, this prevents a Switch from having a tight coupling
// to the command line.
func provideSwitchConfig() fx.Option {
	return fx.Provide(
		func(in SwitchConfigIn) SwitchConfig {
			return SwitchConfig{
				Logger:    in.Logger,
				TTL:       in.CommandLine.TTL,
				MaxMisses: in.CommandLine.Misses,
				Actions:   in.Actions,
				Clock:     in.Clock,
			}
		},
	)
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

	if s.clock == nil {
		s.clock = chronon.SystemClock()
	}

	return s
}

// activate establishes the channels necessary to run this Switch.
// If this switch is already running, and error is returned.
func (s *Switch) activate() (postpone <-chan PostponeRequest, cancel <-chan struct{}, err error) {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()

	if s.cancel == nil {
		s.postpone = make(chan PostponeRequest, 1)
		postpone = s.postpone

		s.cancel = make(chan struct{})
		cancel = s.cancel
	} else {
		err = ErrActive
	}

	return
}

// Activate blocks until either the actions are triggered or Deactivate is invoked.
// If this switch is currently running, this method returns ErrActive.
func (s *Switch) Activate() error {
	postpone, cancel, err := s.activate()
	if err != nil {
		return err
	}

	defer s.Deactivate() // ensure proper cleanup
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
			return ErrDeactivated

		case <-t.C():
			misses++
			s.logger.Printf("missed postpone update [misses=%d]", misses)

			if misses >= s.maxMisses {
				s.logger.Printf("triggering actions")
				Trigger(s.logger, s.actions...)
				return nil // TODO report action errors in return value somehow
			}
		}
	}
}

// Deactivate forces Activate to return without triggering any actions.
// This method returns ErrNotActive if this switch is not active, which
// includes the case where actions have already been triggered.
func (s *Switch) Deactivate() (err error) {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()

	if s.cancel != nil {
		close(s.cancel)
		s.postpone = nil
		s.cancel = nil
	} else {
		err = ErrNotActive
	}

	return
}

// Postpone will delay triggering actions.  The miss count will be reset,
// if applicable.  This method returns true to indicate that actions were
// postponed, false if this switch was not active.
func (s *Switch) Postpone(u PostponeRequest) (postponed bool) {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()

	if s.postpone != nil {
		postponed = true
		s.postpone <- u
	}

	return
}

// provideSwitch creates an fx.Option that fully bootstraps a *Switch component,
// binding it to the fx.App lifecycle.  The only required component is a SwitchConfig,
// typically supplied with provideSwitchConfig.
func provideSwitch() fx.Option {
	return fx.Options(
		fx.Provide(
			NewSwitch,
			func(s *Switch) Postponer {
				return s
			},
		),
		fx.Invoke(
			func(l fx.Lifecycle, s *Switch) {
				l.Append(fx.Hook{
					OnStart: func(context.Context) error {
						go s.Activate()
						return nil
					},
					OnStop: func(context.Context) error {
						return s.Deactivate()
					},
				})
			},
		),
	)
}
