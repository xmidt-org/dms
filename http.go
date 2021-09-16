package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"go.uber.org/fx"
)

const (
	SourceParameter = "source"
)

type PostponeHandler struct {
	Postponer Postponer
}

func (ph PostponeHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	err := request.ParseForm()
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
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
			func(p Postponer) *mux.Router {
				r := mux.NewRouter()
				r.Handle("/postpone", PostponeHandler{Postponer: p}).Methods("PUT")
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
			func(l fx.Lifecycle, s fx.Shutdowner, logger Logger, server *http.Server) {
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
						logger.Printf("PUT http://%s/postpone to postpone triggering actions", server.Addr)

						go func() {
							defer s.Shutdown()
							err := server.Serve(l)
							if !errors.Is(err, http.ErrServerClosed) {
								logger.Printf("HTTP server error: %s", err)
							}
						}()

						return nil
					},
					OnStop: server.Shutdown,
				})
			},
		),
	)
}
