package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"go.uber.org/fx"
)

func run(args []string) error {
	cl, err := parseCommandLine(args)
	if err != nil {
		return err
	}

	var debug Logger
	if cl.Debug {
		debug = WriterLogger{Writer: os.Stdout}
	} else {
		debug = WriterLogger{Writer: io.Discard}
	}

	app := fx.New(
		fx.Logger(debug),
		fx.Supply(cl),
		fx.Provide(
			func() Logger {
				return WriterLogger{Writer: os.Stdout}
			},
			func(cl *CommandLine, s fx.Shutdowner) (actions []Action, err error) {
				actions, err = ParseExec(cl)
				if err == nil {
					actions = append(actions, ShutdownerAction{Shutdowner: s})
				}

				return
			},
			func(cl *CommandLine, s fx.Shutdowner, l Logger, actions []Action) *Switch {
				return NewSwitch(l, cl.TTL, cl.Misses, actions...)
			},
			func(s *Switch) SwitchHandler {
				return SwitchHandler{
					Switch: s,
				}
			},
			func(cl *CommandLine, h SwitchHandler) *http.Server {
				address := cl.Listen

				// just a port is allowed
				p, err := strconv.Atoi(cl.Listen)
				if err == nil {
					address = fmt.Sprintf(":%d", p)
				}

				r := mux.NewRouter()
				r.Handle("/postpone", h).Methods("PUT")

				return &http.Server{
					Addr:    address,
					Handler: r,
				}
			},
		),
		fx.Invoke(
			func(l fx.Lifecycle, s fx.Shutdowner, server *http.Server) {
				l.Append(fx.Hook{
					OnStart: func(context.Context) error {
						go func() {
							defer s.Shutdown()
							server.ListenAndServe()
						}()

						return nil
					},
					OnStop: server.Shutdown,
				})
			},
			func(l fx.Lifecycle, s *Switch) {
				l.Append(fx.Hook{
					OnStart: s.Start,
					OnStop:  s.Stop,
				})
			},
		),
	)

	app.Run()
	return app.Err()
}

func main() {
	err := run(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
