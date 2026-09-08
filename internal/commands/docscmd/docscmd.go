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
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

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
	shouldRun, err := hasExampleMarkers(app.CWD)
	if err != nil {
		return err
	}
	if !shouldRun {
		return nil
	}

	args := []string{"--root", app.CWD}
	if check {
		args = append(args, "--check")
	}

	return app.RunTool(ctx, "docsync", args)
}

func hasExampleMarkers(root string) (bool, error) {
	readmePath := filepath.Join(root, "README.md")
	readmeHasMarker, err := markdownFileHasExampleMarker(readmePath)
	if err != nil {
		return false, err
	}
	if readmeHasMarker {
		return true, nil
	}

	docsDir := filepath.Join(root, "docs")
	info, err := os.Stat(docsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}
	if !info.IsDir() {
		return false, nil
	}

	walkErr := filepath.WalkDir(docsDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(path), ".md") {
			return nil
		}

		//nolint:gosec // path comes from WalkDir rooted at docsDir.
		hasMarker, err := markdownFileHasExampleMarker(path)
		if err != nil {
			return err
		}
		if hasMarker {
			return fs.SkipAll
		}

		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, fs.SkipAll) {
		return false, walkErr
	}

	return errors.Is(walkErr, fs.SkipAll), nil
}

func markdownFileHasExampleMarker(path string) (bool, error) {
	if !strings.EqualFold(filepath.Ext(path), ".md") {
		return false, nil
	}

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	//nolint:gosec // path comes from fixed root/docs traversal or root README.
	raw, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	return strings.Contains(string(raw), "<!-- EXAMPLE:"), nil
}
