package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"go.uber.org/fx"
)

const (
	// SourceParameter is the name of the HTTP query or form parameter identifying the
	// remote entity that is postponing action triggers.
	SourceParameter = "source"

	// PostponePath is the URI path for the postpone handler.
	PostponePath = "/postpone"
)

type notFoundHandler struct {
	l Logger
}

func (nfh notFoundHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	nfh.l.Printf("%s %s NOT FOUND", request.Method, request.RequestURI)
	response.WriteHeader(http.StatusNotFound)
}

type methodNotAllowedHandler struct {
	l Logger
}

func (mnah methodNotAllowedHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	mnah.l.Printf("%s %s METHOD NOT ALLOWED", request.Method, request.RequestURI)
	response.WriteHeader(http.StatusMethodNotAllowed)
}

type PostponeHandler struct {
	Postponer Postponer
}

func (ph PostponeHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	err := request.ParseForm()
	if err != nil {
		response.WriteHeader(http.StatusBadRequest)
		response.Write([]byte(err.Error()))
		return
	}

	pr := PostponeRequest{
		Source:     request.Form.Get(SourceParameter),
		RemoteAddr: request.RemoteAddr,
	}

	if ph.Postponer.Postpone(pr) {
		response.WriteHeader(http.StatusOK)
	} else {
		response.WriteHeader(http.StatusServiceUnavailable)
	}
}

func provideHTTP() fx.Option {
	return fx.Options(
		fx.Provide(
			func(logger Logger, p Postponer) *mux.Router {
				r := mux.NewRouter()
				r.Handle(PostponePath, PostponeHandler{Postponer: p}).Methods("PUT")
				r.NotFoundHandler = notFoundHandler{l: logger}
				r.MethodNotAllowedHandler = methodNotAllowedHandler{l: logger}

				return r
			},
			func(cl CommandLine, r *mux.Router) *http.Server {
				address := cl.HTTP

				// just a port is allowed
				p, err := strconv.Atoi(address)
				if err == nil {
					address = fmt.Sprintf(":%d", p)
				}

				return &http.Server{
					Addr:    address,
					Handler: r,
				}
			},
		),
		fx.Invoke(
			func(logger Logger, l fx.Lifecycle, s fx.Shutdowner, server *http.Server) {
				l.Append(fx.Hook{
					OnStart: func(ctx context.Context) error {
						var lc net.ListenConfig
						l, err := lc.Listen(ctx, "tcp", server.Addr)
						if err != nil {
							return err
						}

						// update the server with the actual listen address
						// this handles cases where port 0 is used to bind to the first available port
						server.Addr = l.Addr().String()
						logger.Printf("PUT http://%s%s to postpone triggering actions", server.Addr, PostponePath)

						go func() {
							defer s.Shutdown()
							logServerError(logger, server.Serve(l))
						}()

						return nil
					},
					OnStop: server.Shutdown,
				})
			},
		),
	)
}
