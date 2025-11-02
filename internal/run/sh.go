package run

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/alexjoedt/forge/internal/log"
)

// Result holds the output and exit status of a command execution.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
}

// Cmd executes a command with the given context and returns stdout, stderr, and exit code.
// The command is logged at debug level before execution.
func Cmd(ctx context.Context, name string, args ...string) Result {
	logger := log.FromContext(ctx)
	logger.Debugf("executing command: %s %v", name, args)

	cmd := exec.CommandContext(ctx, name, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
		Err:      err,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			// Command couldn't be started or other error
			result.ExitCode = -1
		}
		logger.Debugf("command failed: %s (exit code: %d, stderr: %s)", name, result.ExitCode, result.Stderr)
	} else {
		logger.Debugf("command succeeded: %s (stdout: %s)", name, result.Stdout)
	}

	return result
}

// CmdInDir executes a command in the specified directory.
func CmdInDir(ctx context.Context, dir, name string, args ...string) Result {
	logger := log.FromContext(ctx)
	logger.Debugf("executing command in directory %s: %s %v", dir, name, args)

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
		Err:      err,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
		logger.Debugf("command failed: %s (exit code: %d, stderr: %s)", name, result.ExitCode, result.Stderr)
	}

	return result
}

// MustSucceed wraps a Result and returns an error if the command failed.
func (r Result) MustSucceed(cmdDesc string) error {
	if r.Err != nil {
		if r.Stderr != "" {
			return fmt.Errorf("%s failed: %w\nstderr: %s", cmdDesc, r.Err, r.Stderr)
		}
		return fmt.Errorf("%s failed: %w", cmdDesc, r.Err)
	}
	return nil
}

// Success returns true if the command executed successfully (exit code 0).
func (r Result) Success() bool {
	return r.ExitCode == 0 && r.Err == nil
}
