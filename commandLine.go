package main

import (
	"os"
	"time"

	"github.com/alecthomas/kong"
	"go.uber.org/fx"
)

type CommandLine struct {
	Exec   []string      `name:"exec" short:"e" required:"" help:"one or more commands to execute when the switch triggers"`
	Dir    string        `name:"dir" short:"d" optional:"" help:"the working directory for all commands"`
	HTTP   string        `name:"http" short:"h" default:":8080" help:"the HTTP listen address or port"`
	TTL    time.Duration `name:"ttl" short:"t" default:"1m" help:"the maximum interval for TTL updates to keep the switch open"`
	Misses int           `name:"misses" short:"m" default:"1" help:"the maximum number of missed updates allowed before the switch closes"`
	Debug  bool          `name:"debug" default:"false" help:"produce debug logging"`
}

func parseCommandLine(args []string) fx.Option {
	var (
		options []fx.Option
		cl      CommandLine
		k, err  = kong.New(
			&cl,
			kong.Description(
				"A dead man's switch which invokes one or more actions unless postponed on regular intervals.  To postpone the action(s), issue an HTTP PUT to /postpone, with no body, to the configured HTTP address.",
			),
		)
	)

	if err == nil {
		_, err = k.Parse(args)
	}

	if err == nil {
		var debug Logger
		if cl.Debug {
			debug = WriterLogger{Writer: os.Stdout}
		} else {
			debug = DiscardLogger{}
		}

		options = append(options,
			fx.Logger(debug),
			fx.Supply(cl),
			fx.Provide(),
		)
	}

	if err != nil {
		options = append(options, fx.Error(err))
	}

	return fx.Options(options...)
}
