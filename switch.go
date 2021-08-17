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
	DefaultTTL    time.Duration = 1 * time.Minute
	DefaultMisses               = 0
)

var (
	ErrSwitchStarted = errors.New("That switch has already been started")
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

func (pr PostponeRequest) String() string {
	source := pr.Source
	if len(source) == 0 {
		source = "<unset>"
	}

	if len(pr.RemoteAddr) > 0 {
		return fmt.Sprintf("[source=%s] [remoteaddr=%s]", source, pr.RemoteAddr)
	} else {
		return fmt.Sprintf("[source=%s]", source)
	}
}

type Postponer interface {
	Postpone(PostponeRequest) bool
}

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

func (s *Switch) loop(postpone <-chan PostponeRequest, cancel <-chan struct{}) {
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

func (s *Switch) Stop(context.Context) error {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()

	if s.cancel == nil {
		return ErrSwitchStopped
	}

	close(s.cancel)
	s.postpone = nil
	s.cancel = nil
	return nil
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
