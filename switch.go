package main

import (
	"context"
	"errors"
	"sync"
	"time"
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

type PostponeRequest struct {
	Source string
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
	s.logger.Printf("starting switch loop")
	var misses int

	for {
		timer, stop := s.newTimer(s.ttl)
		select {
		case pr := <-postpone:
			stop()
			misses = 0
			s.logger.Printf("postponed [source=%s]", pr.Source)

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
					if err := a.Execute(); err != nil {
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
