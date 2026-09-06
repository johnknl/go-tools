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

// Package releasecmd implements release command workflows.
package releasecmd

import (
	"strings"

	"github.com/johnknl/go-tools/internal/runtime/appctx"
	"github.com/johnknl/go-tools/internal/runtime/execx"
	"github.com/spf13/cobra"
)

// NewCommand builds the release root command.
func NewCommand(app *appctx.Context) *cobra.Command {
	cmd := &cobra.Command{Use: "release", Short: "Release workflows"}

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create release tag and push",
		RunE: func(cmd *cobra.Command, _ []string) error {
			currentRaw, err := app.RunToolOutput(cmd.Context(), "svu", []string{"current"})
			if err != nil {
				return err
			}
			nextRaw, err := app.RunToolOutput(cmd.Context(), "svu", []string{"next"})
			if err != nil {
				return err
			}
			current := strings.TrimSpace(currentRaw)
			next := strings.TrimSpace(nextRaw)
			if err := app.RunTool(cmd.Context(), "gorelease", []string{"-base=" + current, "-version=" + next}); err != nil {
				return err
			}

			tagOpts := execx.RunOptions{Dir: app.CWD, Stdout: app.Stdout, Stderr: app.Stderr}
			if err := app.Runner.Run(cmd.Context(), "git", []string{"tag", next}, tagOpts); err != nil {
				return err
			}
			return app.Runner.Run(cmd.Context(), "git", []string{"push", "origin", next}, tagOpts)
		},
	}

	pkgsiteRefreshCmd := &cobra.Command{
		Use:   "pkgsite-refresh",
		Short: "Refresh latest module on proxy",
		RunE: func(cmd *cobra.Command, _ []string) error {
			args := []string{"list", "-m", "github.com/johnknl/alog@latest"}
			opts := execx.RunOptions{
				Dir:    app.CWD,
				Env:    []string{"GOPROXY=https://proxy.golang.org"},
				Stdout: app.Stdout,
				Stderr: app.Stderr,
			}
			return app.Runner.Run(cmd.Context(), "go", args, opts)
		},
	}

	upgradeToolchainCmd := &cobra.Command{
		Use:   "upgrade-toolchain",
		Short: "Upgrade Go toolchain patch version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			args := []string{"get", "toolchain@patch"}
			opts := execx.RunOptions{Dir: app.CWD, Stdout: app.Stdout, Stderr: app.Stderr}
			return app.Runner.Run(cmd.Context(), "go", args, opts)
		},
	}

	cmd.AddCommand(createCmd, pkgsiteRefreshCmd, upgradeToolchainCmd)
	return cmd
}

// Execute runs the release command tree.
func Execute(app *appctx.Context, args []string) error {
	cmd := NewCommand(app)
	cmd.SetArgs(args)
	return cmd.Execute()
}
