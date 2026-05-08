package runcmd

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/grzadr/godlv/internal/config"
	"github.com/grzadr/godlv/internal/setup"
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

func RunCmd(ctx context.Context, app *setup.App, cfg *config.ArgConfig) error {
	app.Info("run", "cfg", cfg)

	defaultArgs, flagErr := config.NewArgFlags(cfg)

	if flagErr != nil {
		return fmt.Errorf("error parsing default args: %w", flagErr)
	}

	app.Info("flags", "arg", defaultArgs)

	cmd := "yt-dlp"

	args := slices.Concat(defaultArgs, cfg.NonFlag)

	_, resultChan, cancel, err := ExecCmd(
		ctx,
		cmd,
		true,
		args...,
	)
	if err != nil {
		return fmt.Errorf("error setting up command: %w", err)
	}
	defer cancel()

	result := <-resultChan
	app.Info("finished", "result", result)
	return result.Err
}

//go:generate stringer -type=JobStatus
type JobStatus int

const (
	StatusQueued JobStatus = iota
	StatusQueueFailed
	StatusQueueRejected
	StatusRejected
	StatusRunning
	StatusFailed
	StatusCompleted
)

type JobMessage struct {
	id        uuid.UUID
	timestamp time.Time
	status    JobStatus
}

type JobSupervisor struct {
	capacity chan struct{}
	wg       *sync.WaitGroup
	msg      chan JobMessage
}

func NewJobSupervisor(capacity int) *JobSupervisor {
	return &JobSupervisor{
		capacity: make(chan struct{}, max(capacity, 1)),
		wg:       new(sync.WaitGroup),
		msg:      make(chan JobMessage, 1),
	}
}

func (js *JobSupervisor) Wait() {
	js.wg.Wait()
}

func (js *JobSupervisor) Go(
	id uuid.UUID,
	ctx context.Context,
	app *setup.App,
	cfg *config.ArgConfig,
) {
	payload := JobMessage{
		id:        id,
		timestamp: time.Now(),
		status:    StatusQueued,
	}
	select {
	case js.msg <- payload:
	case <-ctx.Done():
		return
	}

	err := RunCmd(ctx, app, cfg)
}

func (js *JobSupervisor) Add(
	ctx context.Context,
	app *setup.App,
	cfg *config.ArgConfig,
) (uuid.UUID, error) {
	select {
	case js.capacity <- struct{}{}:
		id, idErr := uuid.NewV7()

		if idErr != nil {
			return uuid.UUID{}, fmt.Errorf("error generating uuid: %w", idErr)
		}

		js.wg.Go(func() { js.Go(id, ctx, app, cfg) })

		return id, nil

	case <-ctx.Done():
		return uuid.UUID{}, nil
	}
}
