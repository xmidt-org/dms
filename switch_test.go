package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestDefaultNewTimer(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		timer, stop = defaultNewTimer(50 * time.Millisecond)
	)

	require.NotNil(timer)
	require.NotNil(stop)

	select {
	case <-timer:
		// passing
	case <-time.After(time.Second):
		assert.Fail("no time sent on default timer channel")
	}

	stop()
}

type SwitchSuite struct {
	suite.Suite
}

func TestSwitch(t *testing.T) {
	suite.Run(t, new(SwitchSuite))
}
