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

// Package toolscmd implements tooling maintenance commands.
package toolscmd

import (
	"github.com/johnknl/go-tools/internal/runtime/appctx"
	"github.com/johnknl/go-tools/internal/runtime/execx"
	"github.com/johnknl/go-tools/internal/runtime/tooling"
	"github.com/spf13/cobra"
)

// NewCommand builds the tools root command.
func NewCommand(app *appctx.Context) *cobra.Command {
	const cmdTidy = "tidy"
	cmd := &cobra.Command{Use: "tools", Short: "Tool dependency workflows"}

	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Declare pinned tools in go.mod",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return tooling.InstallDeclaredTools(cmd.Context(), app.Runner, app.CWD, app.Stdout, app.Stderr)
		},
	}

	tidyCmd := &cobra.Command{
		Use:   cmdTidy,
		Short: "Run go mod tidy and verify",
		RunE: func(cmd *cobra.Command, _ []string) error {
			const modToken = "mod"
			mainOpts := execx.RunOptions{Dir: app.CWD, Stdout: app.Stdout, Stderr: app.Stderr}
			if err := app.Runner.Run(
				cmd.Context(),
				"go",
				[]string{modToken, cmdTidy},
				mainOpts,
			); err != nil {
				return err
			}
			return app.Runner.Run(cmd.Context(), "go", []string{modToken, "verify"}, mainOpts)
		},
	}

	cmd.AddCommand(installCmd, tidyCmd)
	return cmd
}

// Execute runs the tools command tree.
func Execute(app *appctx.Context, args []string) error {
	cmd := NewCommand(app)
	cmd.SetArgs(args)
	return cmd.Execute()
}
