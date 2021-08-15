package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"go.uber.org/fx"
)

type PostponeHandler struct {
	Postponer Postponer
}

func (ph PostponeHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	pr := PostponeRequest{
		Source: request.RemoteAddr,
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
			func(p Postponer) PostponeHandler {
				return PostponeHandler{
					Postponer: p,
				}
			},
			func(cl CommandLine, ph PostponeHandler) *http.Server {
				address := cl.Listen

				// just a port is allowed
				p, err := strconv.Atoi(cl.Listen)
				if err == nil {
					address = fmt.Sprintf(":%d", p)
				}

				r := mux.NewRouter()
				r.Handle("/postpone", ph).Methods("PUT")

				return &http.Server{
					Addr:    address,
					Handler: r,
				}
			},
		),
		fx.Invoke(
			func(l fx.Lifecycle, s fx.Shutdowner, logger Logger, server *http.Server) {
				l.Append(fx.Hook{
					OnStart: func(context.Context) error {
						go func() {
							defer s.Shutdown()
							err := server.ListenAndServe()
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
