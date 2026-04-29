package main

import (
	"errors"
	"flag"
	"os"

	"github.com/grzadr/godlv/internal/config"
	"github.com/grzadr/godlv/internal/runcmd"
	"github.com/grzadr/godlv/internal/setup"
	"github.com/grzadr/godlv/internal/web"
)

const (
	exitCodeSuccess = 0
	exitCodeErr     = 2
)

func run(app *setup.App, cfg *config.ArgConfig) error {
	ctx, cancel := setup.NewContext()
	defer cancel()

	if len(cfg.NonFlag) > 0 {
		return runcmd.RunCmd(ctx, app, cfg)
	}
	return web.RunServer(ctx, app, cfg)
}

func main() {
	conf, confErr := config.NewArgConfig(os.Args[1:])
	app := setup.NewApp(conf.LogLevel)

	if confErr != nil {
		if !errors.Is(confErr, flag.ErrHelp) {
			app.Error("error parsing flags", "msg", confErr)
			os.Exit(exitCodeErr)
		}

		os.Exit(exitCodeSuccess)
	}

	if runErr := run(app, conf); runErr != nil {
		app.Error("runtime error", "msg", runErr)
		os.Exit(exitCodeErr)
	}
}
