package main

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/suite"
)

type NewAppSuite struct {
	DMSSuite

	validParameters   [][]string
	invalidParameters [][]string
}

func (suite *NewAppSuite) SetupSuite() {
	suite.validParameters = [][]string{
		{"--exec", "echo 'hi'"},
		{"--exec", "echo 'hi'", "--ttl", "10s", "--misses", "4"},
		{"--exec", "echo 'hi'", "--exec", "echo 'another'", "--ttl", "12h", "--misses", "2", "--debug"},
	}

	suite.invalidParameters = [][]string{
		{},
		{"--foobar"},
		{"--exec", "echo 'hi'", "--foobar"},
	}
}

func (suite *NewAppSuite) TestNewApp() {
	suite.Run("Valid", func() {
		for i, validParameters := range suite.validParameters {
			suite.Run(strconv.Itoa(i), func() {
				app := newApp(validParameters)
				suite.Require().NotNil(app)
				suite.NoError(app.Err())
			})
		}
	})

	suite.Run("Invalid", func() {
		for i, invalidParameters := range suite.invalidParameters {
			suite.Run(strconv.Itoa(i), func() {
				app := newApp(invalidParameters)
				suite.Require().NotNil(app)
				suite.Error(app.Err())
			})
		}
	})
}

func TestNewApp(t *testing.T) {
	suite.Run(t, new(NewAppSuite))
}
