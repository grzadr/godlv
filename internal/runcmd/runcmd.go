package runcmd

import (
	"bufio"
	"bytes"
	"context"
	"errors"
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
