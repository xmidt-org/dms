package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type PostponeHandlerSuite struct {
	DMSSuite
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

	p.ExpectPostpone(
		PostponeRequest{Source: source, RemoteAddr: remoteAddr},
	).Return(true).Once()
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

	p.ExpectPostpone(
		PostponeRequest{Source: source, RemoteAddr: remoteAddr},
	).Return(false).Once()
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

type ProvideHTTPSuite struct {
	DMSSuite
}

// newApp creates an *fx.App for tests where the container startup should fail
func (suite *ProvideHTTPSuite) newBadApp(cl CommandLine, p Postponer) {
	app := fx.New(
		fx.Logger(DiscardLogger{}),
		suite.provideLogger(),
		fx.Supply(cl, p),
		provideHTTP(),
		fx.Provide(
			func() Postponer { return p }, // have to do this to ensure the component is of the interface type
		),
	)

	defer app.Stop(context.Background())
	suite.Error(app.Start(context.Background()))
}

// newTestApp creates an *fxtest.App for tests where the container should start just fine
func (suite *ProvideHTTPSuite) newTestApp(cl CommandLine, p Postponer, s **http.Server) *fxtest.App {
	return fxtest.New(
		suite.T(),
		fx.Logger(DiscardLogger{}),
		suite.provideLogger(),
		fx.Supply(cl, p),
		provideHTTP(),
		fx.Provide(
			func() Postponer { return p }, // have to do this to ensure the component is of the interface type
		),
		fx.Populate(s),
	)
}

func (suite *ProvideHTTPSuite) testPostponed(cl CommandLine) {
	var (
		p   = new(mockPostponer)
		s   *http.Server
		app = suite.newTestApp(cl, p, &s)
	)

	app.RequireStart()
	suite.Require().NotNil(s)
	suite.Require().NotEmpty(s.Addr)

	p.ExpectPostpone(mock.MatchedBy(func(pr PostponeRequest) bool {
		suite.Empty(pr.Source)
		suite.NotEmpty(pr.RemoteAddr)
		return true
	})).Once().Return(true)

	target := fmt.Sprintf("http://%s%s", s.Addr, PostponePath)
	request, err := http.NewRequest("PUT", target, nil)
	suite.Require().NoError(err)

	response, err := http.DefaultClient.Do(request)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	defer response.Body.Close()
	suite.Equal(http.StatusOK, response.StatusCode)

	body, err := ioutil.ReadAll(response.Body)
	suite.Require().NoError(err)
	suite.Empty(string(body))

	app.RequireStop()
	p.AssertExpectations(suite.T())
}

func (suite *ProvideHTTPSuite) TestPostponed() {
	suite.Run("Default", func() {
		suite.testPostponed(CommandLine{})
	})

	suite.Run("IntegerPort", func() {
		suite.testPostponed(CommandLine{
			HTTP: "0",
		})
	})

	suite.Run("BindAddress", func() {
		suite.testPostponed(CommandLine{
			HTTP: "localhost:0",
		})
	})
}

func (suite *ProvideHTTPSuite) TestNotFound() {
	var (
		p   = new(mockPostponer)
		s   *http.Server
		app = suite.newTestApp(CommandLine{}, p, &s)
	)

	app.RequireStart()
	suite.Require().NotNil(s)
	suite.Require().NotEmpty(s.Addr)

	target := fmt.Sprintf("http://%s/nosuch", s.Addr)
	request, err := http.NewRequest("PUT", target, nil)
	suite.Require().NoError(err)

	response, err := http.DefaultClient.Do(request)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	io.Copy(io.Discard, response.Body)
	response.Body.Close()

	suite.Equal(http.StatusNotFound, response.StatusCode)

	app.RequireStop()
	p.AssertExpectations(suite.T())
}

func (suite *ProvideHTTPSuite) TestMethodNotAllowed() {
	var (
		p   = new(mockPostponer)
		s   *http.Server
		app = suite.newTestApp(CommandLine{}, p, &s)
	)

	app.RequireStart()
	suite.Require().NotNil(s)
	suite.Require().NotEmpty(s.Addr)

	target := fmt.Sprintf("http://%s%s", s.Addr, PostponePath)
	request, err := http.NewRequest("GET", target, nil)
	suite.Require().NoError(err)

	response, err := http.DefaultClient.Do(request)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	io.Copy(io.Discard, response.Body)
	response.Body.Close()

	suite.Equal(http.StatusMethodNotAllowed, response.StatusCode)

	app.RequireStop()
	p.AssertExpectations(suite.T())
}

func (suite *ProvideHTTPSuite) TestListenError() {
	// force a bind error by grabbing a port
	var lc net.ListenConfig
	bound, err := lc.Listen(context.Background(), "tcp", ":0")
	suite.Require().NoError(err)
	defer bound.Close()

	p := new(mockPostponer)
	suite.newBadApp(
		CommandLine{HTTP: bound.Addr().String()},
		p,
	)

	p.AssertExpectations(suite.T())
}

func TestProvideHTTP(t *testing.T) {
	suite.Run(t, new(ProvideHTTPSuite))
}
