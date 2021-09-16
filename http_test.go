package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"
)

type PostponeHandlerSuite struct {
	suite.Suite
}

func (suite *PostponeHandlerSuite) newRequest(source, remoteAddr string, body io.Reader) *http.Request {
	target := "/test-postpone"
	if len(source) > 0 {
		target = fmt.Sprintf("%s?%s=%s", target, SourceParameter, source)
	}

	r, err := http.NewRequest("PUT", target, body)
	suite.Require().NoError(err)
	suite.Require().NotNil(r)
	r.RemoteAddr = remoteAddr
	return r
}

func (suite *PostponeHandlerSuite) testPostponed(source, remoteAddr string) {
	var (
		p  = new(mockPostponer)
		ph = PostponeHandler{
			Postponer: p,
		}

		// ParseForm needs a body to succeed, even if it's empty
		request  = suite.newRequest(source, remoteAddr, new(bytes.Buffer))
		response = httptest.NewRecorder()
	)

	p.On("Postpone", PostponeRequest{Source: source, RemoteAddr: remoteAddr}).Once().Return(true)
	ph.ServeHTTP(response, request)

	result := response.Result()
	suite.Equal(http.StatusOK, result.StatusCode)

	body, err := ioutil.ReadAll(result.Body)
	suite.NoError(err)
	suite.Empty(string(body)) // string errors are easier to debug

	p.AssertExpectations(suite.T())
}

func (suite *PostponeHandlerSuite) TestPostponed() {
	for _, source := range []string{"", "test"} {
		for _, remoteAddr := range []string{"", "1.1.1.1"} {
			suite.Run(fmt.Sprintf("source=%s, remoteAddr=%s", source, remoteAddr), func() {
				suite.testPostponed(source, remoteAddr)
			})
		}
	}
}

func (suite *PostponeHandlerSuite) testAlreadyTriggered(source, remoteAddr string) {
	var (
		p  = new(mockPostponer)
		ph = PostponeHandler{
			Postponer: p,
		}

		// ParseForm needs a body to succeed, even if it's empty
		request  = suite.newRequest(source, remoteAddr, new(bytes.Buffer))
		response = httptest.NewRecorder()
	)

	p.On("Postpone", PostponeRequest{Source: source, RemoteAddr: remoteAddr}).Once().Return(false)
	ph.ServeHTTP(response, request)

	result := response.Result()
	suite.Equal(http.StatusServiceUnavailable, result.StatusCode)

	body, err := ioutil.ReadAll(result.Body)
	suite.NoError(err)
	suite.Empty(string(body)) // string errors are easier to debug

	p.AssertExpectations(suite.T())
}

func (suite *PostponeHandlerSuite) TestAlreadyTriggered() {
	for _, source := range []string{"", "test"} {
		for _, remoteAddr := range []string{"", "1.1.1.1"} {
			suite.Run(fmt.Sprintf("source=%s, remoteAddr=%s", source, remoteAddr), func() {
				suite.testAlreadyTriggered(source, remoteAddr)
			})
		}
	}
}

func (suite *PostponeHandlerSuite) TestParseFormError() {
	var (
		p  = new(mockPostponer)
		ph = PostponeHandler{
			Postponer: p,
		}

		// ParseForm needs a body to succeed, and we want it to fail
		request  = suite.newRequest("fail", "", nil)
		response = httptest.NewRecorder()
	)

	ph.ServeHTTP(response, request)

	result := response.Result()
	suite.Equal(http.StatusBadRequest, result.StatusCode)

	body, err := ioutil.ReadAll(result.Body)
	suite.NoError(err)
	suite.NotEmpty(string(body)) // string errors are easier to debug

	p.AssertExpectations(suite.T())
}

func TestPostponeHandler(t *testing.T) {
	suite.Run(t, new(PostponeHandlerSuite))
}
