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

// Package appctx wires shared runtime dependencies for commands.
package appctx

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/johnknl/go-tools/internal/runtime/execx"
	"github.com/johnknl/go-tools/internal/runtime/prompt"
	"github.com/johnknl/go-tools/internal/runtime/tooling"
)

// Context stores process and runtime dependencies shared across commands.
//
//nolint:govet // keeping explicit runtime fields grouped for command wiring clarity.
type Context struct {
	Stdout io.Writer
	Stderr io.Writer
	CWD    string
	Stdin  *os.File

	Runner execx.Runner

	Yes     bool
	NoInput bool
}

// New constructs a Context rooted at cwd.
func New(cwd string) Context {
	runner := execx.NewRunner()
	return Context{
		CWD:    cwd,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Runner: runner,
	}
}

// Prompt returns a configured confirmation prompter.
func (c Context) Prompt() prompt.Prompter {
	return prompt.Prompter{In: c.Stdin, Out: c.Stdout, Yes: c.Yes, NoInput: c.NoInput}
}

// RunTool runs a named external tool.
func (c Context) RunTool(ctx context.Context, name string, args []string) error {
	if err := tooling.EnsureTool(ctx, c.Runner, c.CWD, name, c.Stdout, c.Stderr); err != nil {
		return err
	}

	modFile := filepath.Join(c.CWD, "tools.mod")
	if _, err := os.Stat(modFile); err != nil {
		return fmt.Errorf("missing tools.mod after tool setup: %w", err)
	}
	runArgs := tooling.ToolInvocationArgs(modFile, name, args)
	opts := execx.RunOptions{Dir: c.CWD, Stdout: c.Stdout, Stderr: c.Stderr, Stdin: c.Stdin}
	return c.Runner.Run(ctx, "go", runArgs, opts)
}

// RunToolOutput runs a named external tool and captures stdout.
func (c Context) RunToolOutput(ctx context.Context, name string, args []string) (string, error) {
	if err := tooling.EnsureTool(ctx, c.Runner, c.CWD, name, c.Stdout, c.Stderr); err != nil {
		return "", err
	}

	modFile := filepath.Join(c.CWD, "tools.mod")
	if _, err := os.Stat(modFile); err != nil {
		return "", fmt.Errorf("missing tools.mod after tool setup: %w", err)
	}
	runArgs := tooling.ToolInvocationArgs(modFile, name, args)
	opts := execx.RunOptions{Dir: c.CWD, Stdout: c.Stdout, Stderr: c.Stderr, Stdin: c.Stdin}
	return c.Runner.Output(ctx, "go", runArgs, opts)
}
