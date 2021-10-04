package main

import (
	"fmt"
	"os"

	"github.com/xmidt-org/chronon"
	"go.uber.org/fx"
)

func run(args []string) error {
	app := fx.New(
		parseCommandLine(args),
		provideLogger(os.Stdout),
		fx.Supply(chronon.SystemClock()),
		provideActions(),
		provideSwitch(),
		provideHTTP(),
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
