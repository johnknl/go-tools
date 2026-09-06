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

// Package lintcmd implements lint command workflows.
package lintcmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/johnknl/go-tools/internal/runtime/appctx"
	"github.com/johnknl/go-tools/internal/runtime/execx"
	"github.com/johnknl/go-tools/internal/toolchain"
	"github.com/spf13/cobra"
)

type options struct {
	License   string
	BuildTags string
	Yes       bool
	NoInput   bool
}

const subcommandRun = "run"

// NewCommand builds the lint root command.
func NewCommand(app *appctx.Context) *cobra.Command {
	opts := &options{Yes: app.Yes, NoInput: app.NoInput, BuildTags: toolchain.DefaultBuildTags, License: "LICENSE"}
	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Lint and formatting workflows",
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			app.Yes = opts.Yes
			app.NoInput = opts.NoInput
		},
	}
	cmd.PersistentFlags().BoolVar(&opts.Yes, "yes", opts.Yes, "assume yes for prompts")
	cmd.PersistentFlags().BoolVar(&opts.NoInput, "no-input", opts.NoInput, "disable interactive prompts")
	cmd.PersistentFlags().StringVar(&opts.License, "license-file", opts.License, "license file for addlicense")

	headersCmd := &cobra.Command{
		Use:   "headers",
		Short: "Add/update license headers",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runHeaders(cmd.Context(), app, opts.License, false)
		},
	}
	headersCheckCmd := &cobra.Command{
		Use:   "headers-check",
		Short: "Verify license headers",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runHeaders(cmd.Context(), app, opts.License, true)
		},
	}
	runCmd := &cobra.Command{
		Use:   subcommandRun,
		Short: "Run lint checks",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := runHeaders(cmd.Context(), app, opts.License, true); err != nil {
				return err
			}
			verifyArgs := []string{"config", "verify", "-c", ".golangci.yml"}
			if err := app.RunTool(cmd.Context(), "golangci-lint", verifyArgs); err != nil {
				return err
			}
			return app.RunTool(cmd.Context(), "golangci-lint", []string{"run"})
		},
	}
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runCmd.RunE(cmd, nil)
	}
	fixCmd := &cobra.Command{
		Use:   "fix",
		Short: "Apply formatting and lint fixes",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := runHeaders(cmd.Context(), app, opts.License, false); err != nil {
				return err
			}
			fmtArgs := []string{"fmt", "./..."}
			fmtOpts := execx.RunOptions{
				Dir:    app.CWD,
				Env:    []string{"GOFLAGS=-tags=" + opts.BuildTags},
				Stdout: app.Stdout,
				Stderr: app.Stderr,
			}
			if err := app.Runner.Run(cmd.Context(), "go", fmtArgs, fmtOpts); err != nil {
				return err
			}
			return app.RunTool(cmd.Context(), "golangci-lint", []string{"run", "--fix"})
		},
	}
	fixCmd.Flags().StringVar(&opts.BuildTags, "build-tags", opts.BuildTags, "GOFLAGS build tags for go fmt")
	vulnCmd := &cobra.Command{
		Use:   "vuln",
		Short: "Run govulncheck",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return app.RunTool(cmd.Context(), "govulncheck", []string{"./..."})
		},
	}

	cmd.AddCommand(runCmd, fixCmd, headersCmd, headersCheckCmd, vulnCmd)
	return cmd
}

// Execute runs the lint command tree.
func Execute(app *appctx.Context, args []string) error {
	cmd := NewCommand(app)
	cmd.SetArgs(args)
	return cmd.Execute()
}

func runHeaders(ctx context.Context, app *appctx.Context, license string, check bool) error {
	files, err := goFilesFromGit(ctx, app)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}

	args := []string{"-f", license}
	if check {
		args = append(args, "-check")
	}
	args = append(args, files...)
	return app.RunTool(ctx, "addlicense", args)
}

func goFilesFromGit(ctx context.Context, app *appctx.Context) ([]string, error) {
	trackedOut, err := app.Runner.Output(ctx, "git", []string{"ls-files", "*.go"}, execx.RunOptions{Dir: app.CWD})
	if err != nil {
		return nil, err
	}
	untrackedArgs := []string{"ls-files", "--others", "--exclude-standard", "*.go"}
	untrackedOut, err := app.Runner.Output(ctx, "git", untrackedArgs, execx.RunOptions{Dir: app.CWD})
	if err != nil {
		return nil, err
	}

	merged := map[string]struct{}{}
	for _, block := range []string{trackedOut, untrackedOut} {
		for line := range strings.SplitSeq(block, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			path := filepath.Join(app.CWD, line)
			if _, err := os.Stat(path); err == nil {
				merged[line] = struct{}{}
			}
		}
	}

	files := make([]string, 0, len(merged))
	for f := range merged {
		files = append(files, f)
	}
	if len(files) == 0 {
		return files, nil
	}

	if len(files) > 10000 {
		return nil, fmt.Errorf("too many go files for addlicense invocation")
	}

	return files, nil
}
