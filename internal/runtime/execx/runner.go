// MIT License
//
// Copyright (C) 2026 John Kleijn
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE

// Package execx executes child processes with shared options.
package execx

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// RunOptions controls command execution context and streams.
type RunOptions struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Dir    string
	Env    []string
}

// Runner executes commands.
type Runner struct{}

// NewRunner constructs a Runner.
func NewRunner() Runner { return Runner{} }

// Run executes a command and streams output.
//
//nolint:gosec // command and args are intentionally dynamic for CLI orchestration.
func (Runner) Run(ctx context.Context, name string, args []string, opts RunOptions) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = opts.Dir
	cmd.Env = append(os.Environ(), opts.Env...)
	cmd.Stdin = opts.Stdin
	cmd.Stdout = opts.Stdout
	cmd.Stderr = opts.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run %s: %w", cmd.String(), err)
	}

	return nil
}

// Output executes a command and returns stdout.
//
//nolint:gosec // command and args are intentionally dynamic for CLI orchestration.
func (Runner) Output(ctx context.Context, name string, args []string, opts RunOptions) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = opts.Dir
	cmd.Env = append(os.Environ(), opts.Env...)
	cmd.Stdin = opts.Stdin

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("run %s: %w (%s)", cmd.String(), err, stderr.String())
	}

	return stdout.String(), nil
}
