package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/exec"

	"github.com/grzadr/godlv/internal/app"
	"github.com/grzadr/godlv/internal/config"
	"github.com/grzadr/godlv/internal/runcmd"
)

const (
	exitCode    = 0
	exitCodeErr = 2
)

func run(ctx context.Context, app *app.App, _ *config.ArgConfig) error {
	// cmd :=
	// args := []string{
	// 	"--force-overwrites",
	// 	"--no-progress",
	// 	"-t",
	// 	"mkv",
	// 	"https://www.cda.pl/video/2142915386",
	// }

	cmd := "bash"
	args := []string{
		"-c",
		`for x in $(seq 1 3); do sleep 1; echo "number"; done; exit 0`,
	}

	stdout, resultChan, cancel, err := runcmd.ExecCmd(
		ctx,
		cmd,
		args...)
	if err != nil {
		return err
	}
	defer cancel()
	go func() {
		for msg := range stdout {
			app.Info("received message", "msg", msg)
			// cancel()
			// break
		}
	}()

	result := <-resultChan
	app.Info("finished", "result", result)
	app.Info("error", "canceled", errors.Is(result.Err, context.Canceled))

	if exitErr, ok := errors.AsType[*exec.ExitError](result.Err); ok {
		app.Info("error", "exit_error", exitErr)
	}

	return nil
}

func main() {
	conf, confErr := config.NewArgConfig(os.Args[1:])
	app := app.NewApp(conf.LogLevel)
	ctx := context.Background()

	if confErr != nil {
		if !errors.Is(confErr, flag.ErrHelp) {
			app.Error("error parsing flags", "msg", confErr)
			os.Exit(exitCodeErr)
		}

		os.Exit(exitCode)
	}

	if runErr := run(ctx, app, conf); runErr != nil {
		app.Error("runtime error", "msg", runErr)
		os.Exit(exitCodeErr)
	}
}
