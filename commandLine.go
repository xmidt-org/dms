package main

import (
	"time"

	"github.com/alecthomas/kong"
)

type CommandLine struct {
	Exec   []string      `name:"exec" short:"e" required:"" help:"one or more commands to execute when the switch triggers"`
	Dir    string        `name:"dir" short:"d" optional:"" help:"the working directory for the command"`
	Listen string        `name:"listen" short:"l" default:":8080" help:"the listen address or port"`
	TTL    time.Duration `name:"ttl" short:"t" default:"1m" help:"the maximum interval for TTL updates to keep the switch open"`
	Misses int           `name:"misses" short:"m" default:"1" help:"the maximum number of missed updates allowed before the switch closes"`
	Debug  bool          `name:"debug" short:"d" default:"false" help:"produce debug logging"`
}

func parseCommandLine(args []string) (cl *CommandLine, err error) {
	cl = new(CommandLine)

	var k *kong.Kong
	k, err = kong.New(
		cl,
		kong.Description("A dead man's switch which invokes an action when a heartbeat stops"),
	)

	if err == nil {
		_, err = k.Parse(args)
	}

	return
}
