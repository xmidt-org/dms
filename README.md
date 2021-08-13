# dms

dms is a command-line dead man's switch that will trigger one or more actions unless postponed.

[![Build Status](https://github.com/xmidt-org/dms/workflows/CI/badge.svg)](https://github.com/xmidt-org/dms/actions)
[![codecov.io](http://codecov.io/github/xmidt-org/dms/coverage.svg?branch=main)](http://codecov.io/github/xmidt-org/dms?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/xmidt-org/dms)](https://goreportcard.com/report/github.com/xmidt-org/dms)
[![Apache V2 License](http://img.shields.io/badge/license-Apache%20V2-blue.svg)](https://github.com/xmidt-org/dms/blob/main/LICENSE)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=xmidt-org_PROJECT&metric=alert_status)](https://sonarcloud.io/dashboard?id=xmidt-org_PROJECT)
[![GitHub release](https://img.shields.io/github/release/xmidt-org/dms.svg)](CHANGELOG.md)

## Summary

dms is a command-line utility that will trigger one or more actions unless postponed by performing an HTTP PUT to its /postpone endpoint.

### Usage
```
dms --help
Usage: dms --exec=EXEC,...

A dead man's switch which invokes one or more actions unless postponed on regular intervals. To postpone the action(s),
issue an HTTP PUT to /postpone, with no body, to the configured listen address.

Flags:
  -h, --help              Show context-sensitive help.
  -e, --exec=EXEC,...     one or more commands to execute when the switch triggers
  -d, --dir=STRING        the working directory for the command
  -l, --listen=":8080"    the listen address or port
  -t, --ttl=1m            the maximum interval for TTL updates to keep the switch open
  -m, --misses=1          the maximum number of missed updates allowed before the switch closes
      --debug             produce debug logging
```

### Actions
Actions are supplied on the command line via `--exec` or `-e`.  At least (1) action is required.  When triggered, each action will be executed one at a time, in the order specified on the command line.  After triggering actions, `dms` will exit.

```
dms --exec "echo 'here is just one action'"
dms --exec "echo '1'" --exec "echo '2'"
```

### Listen address
The `--listen` or `-l` options change the bind address for the HTTP server.  The endpoint is always /postpone at this address.  Either a simple port or a `golang` network address is allowed:

```
dms --exec "echo 'oh noes!'" --listen ":9100"
dms --exec "echo 'oh noes!'" --listen 6600
dms --exec "echo 'oh noes!'" --listen "localhost:11000"
```

If no listen address is supplied, `dms` uses `:8080`.

### TTL
By default, an HTTP PUT must be made to the /postpone endpoint every minute.  This can be changed with `--ttl` or `-t`, passing a string that is in the same format as `golang` durations:

```
dms --exec "echo 'hi there'" --ttl 30s
```

### Misses
`dms` will trigger its actions upon the first missed postpone.  This can be changed with `--misses` or `-m` to allow one or more missed heartbeats.  For example, this will allow (2) missed PUTs before triggering actions:

```
dms --exec "format c:" --misses 2
```

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Details](#details)
- [Install](#install)
- [Contributing](#contributing)

## Code of Conduct

This project and everyone participating in it are governed by the [XMiDT Code Of Conduct](https://xmidt.io/docs/community/code_of_conduct/). 
By participating, you agree to this Code.

## Install

```
go get -u github.com/xmidt-org/dms
```

## Contributing

Refer to [CONTRIBUTING.md](CONTRIBUTING.md).
