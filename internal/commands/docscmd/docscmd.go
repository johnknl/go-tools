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

// Package docscmd implements docs command workflows.
package docscmd

import (
	"context"

	"github.com/johnknl/go-tools/internal/runtime/appctx"
	"github.com/johnknl/go-tools/internal/runtime/execx"
	"github.com/spf13/cobra"
)

// NewCommand builds the docs root command.
func NewCommand(app *appctx.Context) *cobra.Command {
	image := "squidfunk/mkdocs-material:9"
	port := "8000:8000"

	cmd := &cobra.Command{Use: "docs", Short: "Documentation workflows"}

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run docs dev server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := runDocsync(cmd.Context(), app, false); err != nil {
				return err
			}
			args := []string{"run", "--rm", "-p", port, "-v", app.CWD + ":/docs", image}
			opts := execx.RunOptions{
				Dir:    app.CWD,
				Stdout: app.Stdout,
				Stderr: app.Stderr,
				Stdin:  app.Stdin,
			}
			return app.Runner.Run(cmd.Context(), "docker", args, opts)
		},
	}
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runCmd.RunE(cmd, nil)
	}

	syncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync EXAMPLE markers",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDocsync(cmd.Context(), app, false)
		},
	}

	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "Check EXAMPLE markers",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDocsync(cmd.Context(), app, true)
		},
	}

	runCmd.Flags().StringVar(&image, "image", image, "mkdocs docker image")
	runCmd.Flags().StringVar(&port, "port", port, "host:container port mapping")
	cmd.AddCommand(runCmd, syncCmd, checkCmd)

	return cmd
}

// Execute runs the docs command tree.
func Execute(app *appctx.Context, args []string) error {
	cmd := NewCommand(app)
	cmd.SetArgs(args)
	return cmd.Execute()
}

func runDocsync(ctx context.Context, app *appctx.Context, check bool) error {
	args := []string{"--root", app.CWD}
	if check {
		args = append(args, "--check")
	}

	return app.RunTool(ctx, "docsync", args)
}
