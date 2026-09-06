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

// Package configsync ensures local config files match embedded defaults.
package configsync

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/johnknl/go-tools/internal/runtime/prompt"
)

// Spec describes a managed configuration file.
type Spec struct {
	FileName string
	Prompt   string
	Content  []byte
}

// Ensure creates or updates a managed config file in cwd.
//
//nolint:gosec // target path and mode are user-facing CLI configuration files.
func Ensure(spec Spec, cwd string, prompter prompt.Prompter) error {
	path := filepath.Join(cwd, spec.FileName)
	existing, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if writeErr := os.WriteFile(path, spec.Content, 0o644); writeErr != nil {
				return fmt.Errorf("write missing config %s: %w", path, writeErr)
			}
			return nil
		}

		return fmt.Errorf("read config %s: %w", path, err)
	}

	if checksum(existing) == checksum(spec.Content) {
		return nil
	}

	ok, err := prompter.Confirm(spec.Prompt, false)
	if err != nil {
		return fmt.Errorf("prompt overwrite %s: %w", path, err)
	}
	if !ok {
		return nil
	}

	if err := os.WriteFile(path, spec.Content, 0o644); err != nil {
		return fmt.Errorf("overwrite config %s: %w", path, err)
	}

	return nil
}

func checksum(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
