package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"slices"

	"github.com/grzadr/godlv/internal/config"
	"github.com/grzadr/godlv/internal/runcmd"
	"github.com/grzadr/godlv/internal/setup"
	"github.com/grzadr/godlv/internal/web"
)

const (
	exitCodeSuccess = 0
	exitCodeErr     = 2
)

func runCmd(ctx context.Context, app *setup.App, cfg *config.ArgConfig) error {
	app.Info("run", "cfg", cfg)

	defaultArgs, flagErr := config.NewArgFlags(cfg)

	if flagErr != nil {
		return fmt.Errorf("error parsing default ergs: %w", flagErr)
	}

	app.Info("flags", "arg", defaultArgs)

	cmd := "yt-dlp"

	args := slices.Concat(defaultArgs, cfg.NonFlag)

	_, resultChan, cancel, err := runcmd.ExecCmd(
		ctx,
		cmd,
		true,
		args...,
	)
	if err != nil {
		return err
	}
	defer cancel()

	result := <-resultChan
	app.Info("finished", "result", result)
	app.Info("error", "canceled", errors.Is(result.Err, context.Canceled))

	if exitErr, ok := errors.AsType[*exec.ExitError](result.Err); ok {
		app.Info("error", "exit_error", exitErr)
	}

	return nil
}

func run(app *setup.App, cfg *config.ArgConfig) error {
	ctx, cancel := setup.NewContext()
	defer cancel()

	if len(cfg.NonFlag) > 0 {
		return runCmd(ctx, app, cfg)
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
