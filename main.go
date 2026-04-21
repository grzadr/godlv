package main

import (
	"errors"
	"flag"
	"os"

	"github.com/grzadr/godlv/internal/app"
	"github.com/grzadr/godlv/internal/config"
)

const (
	exitCode    = 0
	exitCodeErr = 2
)

type ExecResult struct {
	ExitCode int
	Msg      string
}

func ExecCommand(cmd string, args ...string) (ExecResult, error) {
	return ExecResult{}, nil
}

func run(_ *app.App, _ *config.ArgConfig) error {
	return nil
}

func main() {
	conf, confErr := config.NewArgConfig(os.Args[1:])
	app := app.NewApp(conf.LogLevel)
	if confErr != nil {
		if !errors.Is(confErr, flag.ErrHelp) {
			app.Error("error parsing flags", "msg", confErr)
			os.Exit(exitCodeErr)
		}

		os.Exit(exitCode)
	}

	if runErr := run(app, conf); runErr != nil {
		app.Error("runtime error", "msg", runErr)
		os.Exit(exitCodeErr)
	}

	app.Info("hello world", "conf", *conf)
	app.Debug("debug statement")
	app.Error("test")
}
