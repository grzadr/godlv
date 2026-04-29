package runcmd

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"sync"
)

type ExecResult struct {
	ExitCode int
	Msg      string
	Err      error
}

func ExecCmd(
	ctx context.Context,
	name string,
	ignoreStdout bool,
	args ...string,
) (<-chan string, <-chan ExecResult, context.CancelFunc, error) {
	cmdCtx, cmdCancel := context.WithCancel(ctx)

	cmd := exec.CommandContext(cmdCtx, name, args...)

	var stdoutChan chan string
	var stdoutPipe io.ReadCloser
	var pipeErr error

	if ignoreStdout {
		cmd.Stdout = io.Discard
	} else {
		stdoutPipe, pipeErr = cmd.StdoutPipe()

		if pipeErr != nil {
			cmdCancel()
			return nil, nil, nil, pipeErr
		}
	}

	stderrBuf := new(bytes.Buffer)
	cmd.Stderr = stderrBuf

	if startErr := cmd.Start(); startErr != nil {
		cmdCancel()
		return nil, nil, nil, startErr
	}

	resultChan := make(chan ExecResult, 1)

	var wg sync.WaitGroup
	var scanErr error

	if !ignoreStdout {
		wg.Go(func() {
			defer close(stdoutChan)
			stdoutChan = make(chan string)
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
	}

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
