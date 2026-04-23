package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"flag"
	"os"
	"os/exec"
	"sync"

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
	Err      error
}

func ExecCommand(
	ctx context.Context,
	name string,
	args ...string,
) (<-chan string, <-chan ExecResult, context.CancelFunc, error) {
	cmdCtx, cmdCancel := context.WithCancel(ctx)

	cmd := exec.CommandContext(cmdCtx, name, args...)
	stdoutPipe, pipeErr := cmd.StdoutPipe()

	if pipeErr != nil {
		cmdCancel()
		return nil, nil, nil, pipeErr
	}

	stderrBuf := new(bytes.Buffer)
	cmd.Stderr = stderrBuf

	if startErr := cmd.Start(); startErr != nil {
		cmdCancel()
		return nil, nil, nil, startErr
	}

	stdoutChan := make(chan string)
	resultChan := make(chan ExecResult, 1)

	var wg sync.WaitGroup
	var scanErr error

	wg.Go(func() {
		defer close(stdoutChan)
		scanner := bufio.NewScanner(stdoutPipe)
		const maxBufferSize = 1024 * 1024
		const minBufferSie = 64 * 1024
		scanner.Buffer(make([]byte, minBufferSie), maxBufferSize)

		for scanner.Scan() {
			select {
			case stdoutChan <- scanner.Text():

			case <-cmdCtx.Done():
				return
			}
		}
		scanErr = scanner.Err()
	})

	go func() {
		defer cmdCancel()
		defer close(resultChan)

		wg.Wait()
		cmdErr := cmd.Wait()

		resultChan <- ExecResult{
			ExitCode: cmd.ProcessState.ExitCode(),
			Msg:      stderrBuf.String(),
			Err:      errors.Join(scanErr, cmdErr, cmdCtx.Err()),
		}
	}()

	return stdoutChan, resultChan, cmdCancel, nil
}

func run(ctx context.Context, _ *app.App, _ *config.ArgConfig) error {
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

	app.Info("hello world", "conf", *conf)
	app.Debug("debug statement")
	app.Error("test")
}
