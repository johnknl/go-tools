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

// Package tooling manages tool declarations in module go.mod files.
package tooling

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/johnknl/go-tools/internal/runtime/execx"
	"github.com/johnknl/go-tools/internal/toolchain"
)

const (
	goCmd            = "go"
	toolSubcmd       = "tool"
	getSubcmd        = "get"
	toolFlag         = "-tool"
	goWorkOff        = "GOWORK=off"
	defaultToolsFile = "tools.mod"
	defaultGoVersion = "1.26.7"
	toolsModule      = "tools"
)

// EnsureTool verifies a tool is available through go tool and lazily declares it when missing.
func EnsureTool(
	ctx context.Context,
	runner execx.Runner,
	cwd string,
	name string,
	stdout io.Writer,
	stderr io.Writer,
) error {
	if ToolAvailable(ctx, runner, cwd, name) {
		return nil
	}

	spec, ok := toolchain.FindToolSpec(name)
	if !ok {
		return fmt.Errorf("missing tool %q and no pinned spec available", name)
	}

	if err := InstallTool(ctx, runner, cwd, spec, stdout, stderr); err != nil {
		return err
	}
	if !ToolAvailable(ctx, runner, cwd, name) {
		return fmt.Errorf("tool %q unavailable after install", name)
	}

	return nil
}

// InstallDeclaredTools registers all pinned tools in the module go.mod.
func InstallDeclaredTools(
	ctx context.Context,
	runner execx.Runner,
	cwd string,
	stdout io.Writer,
	stderr io.Writer,
) error {
	modFile, err := ensureToolsMod(cwd)
	if err != nil {
		return err
	}

	for _, spec := range toolchain.ToolSpecs {
		if err := installToolWithModFile(ctx, runner, cwd, modFile, spec, stdout, stderr); err != nil {
			return err
		}
	}

	return nil
}

// InstallTool registers one pinned tool in the module go.mod.
func InstallTool(
	ctx context.Context,
	runner execx.Runner,
	cwd string,
	spec toolchain.ToolSpec,
	stdout io.Writer,
	stderr io.Writer,
) error {
	modFile, err := ensureToolsMod(cwd)
	if err != nil {
		return err
	}

	return installToolWithModFile(ctx, runner, cwd, modFile, spec, stdout, stderr)
}

func installToolWithModFile(
	ctx context.Context,
	runner execx.Runner,
	cwd string,
	modFile string,
	spec toolchain.ToolSpec,
	stdout io.Writer,
	stderr io.Writer,
) error {
	args := []string{getSubcmd, "-modfile=" + modFile, toolFlag, spec.InstallTarget()}
	opts := execx.RunOptions{Dir: cwd, Env: []string{goWorkOff}, Stdout: stdout, Stderr: stderr}
	if err := runner.Run(ctx, goCmd, args, opts); err != nil {
		return fmt.Errorf("declare tool %s in %s: %w", spec.Name, modFile, err)
	}

	return nil
}

// ToolAvailable reports whether go tool can resolve the tool name.
func ToolAvailable(ctx context.Context, runner execx.Runner, cwd string, name string) bool {
	modFile := filepath.Join(cwd, defaultToolsFile)
	if _, err := os.Stat(modFile); err != nil {
		return false
	}

	args := []string{toolSubcmd, "-modfile=" + modFile, "-n", name}
	err := runner.Run(ctx, goCmd, args, execx.RunOptions{Dir: cwd, Env: []string{goWorkOff}})
	return err == nil
}

// ToolInvocationArgs returns go command args for invoking a declared tool.
func ToolInvocationArgs(modFile string, name string, args []string) []string {
	toolArgs := make([]string, 0, 3+len(args)+1)
	toolArgs = append(toolArgs, toolSubcmd)
	toolArgs = append(toolArgs, "-modfile="+modFile)
	toolArgs = append(toolArgs, name)
	toolArgs = append(toolArgs, args...)
	return toolArgs
}

func ensureToolsMod(cwd string) (string, error) {
	modFile := filepath.Join(cwd, defaultToolsFile)
	if _, err := os.Stat(modFile); err == nil {
		return modFile, nil
	}

	content := fmt.Sprintf("module %s\n\ngo %s\n", toolsModule, defaultGoVersion)
	if err := os.WriteFile(modFile, []byte(content), 0o600); err != nil {
		return "", fmt.Errorf("create %s: %w", modFile, err)
	}

	return modFile, nil
}
