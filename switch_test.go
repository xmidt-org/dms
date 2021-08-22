package main

import (
	"strconv"
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

func TestPostponeRequest(t *testing.T) {
	testCases := []struct {
		request  PostponeRequest
		contains []string
	}{
		{
			request:  PostponeRequest{},
			contains: []string{DefaultSource},
		},
		{
			request: PostponeRequest{
				RemoteAddr: "127.0.0.1",
			},
			contains: []string{DefaultSource, "127.0.0.1"},
		},
		{
			request: PostponeRequest{
				Source: "testytest",
			},
			contains: []string{"testytest"},
		},
		{
			request: PostponeRequest{
				Source:     "testytest",
				RemoteAddr: "127.0.0.1",
			},
			contains: []string{"testytest", "127.0.0.1"},
		},
	}

	for i, testCase := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				assert = assert.New(t)
				s      = testCase.request.String()
			)

			assert.NotEmpty(s)
			for _, v := range testCase.contains {
				assert.Contains(s, v)
			}
		})
	}
}

type SwitchSuite struct {
	suite.Suite
}

func TestSwitch(t *testing.T) {
	suite.Run(t, new(SwitchSuite))
}
