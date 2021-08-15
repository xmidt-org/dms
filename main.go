package main

import (
	"fmt"
	"os"

	"go.uber.org/fx"
)

func run(args []string) error {
	app := fx.New(
		parseCommandLine(args),
		provideLogger(),
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
