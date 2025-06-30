// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"

	"go.uber.org/fx"
)

func newApp(args []string) *fx.App {
	return fx.New(
		parseCommandLine(args),
		provideLogger(os.Stdout),
		provideActions(),
		provideSwitchConfig(),
		provideSwitch(),
		provideHTTP(),
	)
}

func main() {
	app := newApp(os.Args[1:])
	app.Run()
	if app.Err() != nil {
		fmt.Fprintf(os.Stderr, "%s\n", app.Err())
		os.Exit(1)
	}
}
