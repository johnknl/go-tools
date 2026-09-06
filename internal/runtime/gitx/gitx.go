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

// Package gitx contains git helpers for command workflows.
package gitx

import (
	"context"
	"fmt"
	"strings"

	"github.com/johnknl/go-tools/internal/runtime/execx"
)

const gitVerifyFlag = "--verify"
const gitRevParseCmd = "rev-parse"

// ResolveCommit resolves a ref to a commit, fetching origin/ref when needed.
func ResolveCommit(ctx context.Context, runner execx.Runner, cwd string, ref string) (string, error) {
	args := []string{gitRevParseCmd, gitVerifyFlag, "--quiet", ref + "^{commit}"}
	out, err := runner.Output(ctx, "git", args, execx.RunOptions{Dir: cwd})
	if err == nil {
		return strings.TrimSpace(out), nil
	}

	fetchArgs := []string{"fetch", "--quiet", "origin", ref}
	if fetchErr := runner.Run(ctx, "git", fetchArgs, execx.RunOptions{Dir: cwd}); fetchErr != nil {
		return "", fmt.Errorf("resolve ref %s: %w", ref, err)
	}

	parseArgs := []string{gitRevParseCmd, gitVerifyFlag, "FETCH_HEAD^{commit}"}
	fetched, parseErr := runner.Output(ctx, "git", parseArgs, execx.RunOptions{Dir: cwd})
	if parseErr != nil {
		return "", fmt.Errorf("resolve fetched ref %s: %w", ref, parseErr)
	}

	return strings.TrimSpace(fetched), nil
}

// HeadCommit returns the current HEAD commit hash.
func HeadCommit(ctx context.Context, runner execx.Runner, cwd string) (string, error) {
	out, err := runner.Output(ctx, "git", []string{gitRevParseCmd, gitVerifyFlag, "HEAD"}, execx.RunOptions{Dir: cwd})
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(out), nil
}

// ShortCommit shortens a commit hash to a fixed display length.
func ShortCommit(ctx context.Context, runner execx.Runner, cwd string, full string, length int) (string, error) {
	args := []string{gitRevParseCmd, fmt.Sprintf("--short=%d", length), full}
	out, err := runner.Output(ctx, "git", args, execx.RunOptions{Dir: cwd})
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(out), nil
}

// IsDirty reports whether the worktree has tracked or untracked changes.
func IsDirty(ctx context.Context, runner execx.Runner, cwd string) (bool, error) {
	args := []string{"status", "--porcelain", "--untracked-files=normal"}
	out, err := runner.Output(ctx, "git", args, execx.RunOptions{Dir: cwd})
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(out) != "", nil
}
