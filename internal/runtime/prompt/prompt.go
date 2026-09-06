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

// Package prompt provides small interactive prompts.
package prompt

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Prompter reads and writes confirmation prompts.
type Prompter struct {
	In  io.Reader
	Out io.Writer

	Yes     bool
	NoInput bool
}

// Confirm prompts the user for a yes/no answer.
func (p Prompter) Confirm(label string, defaultYes bool) (bool, error) {
	if p.Yes {
		return true, nil
	}
	if p.NoInput {
		return defaultYes, nil
	}

	yn := "[y/N]"
	if defaultYes {
		yn = "[Y/n]"
	}

	if _, err := fmt.Fprintf(p.Out, "%s %s ", label, yn); err != nil {
		return false, err
	}

	r := bufio.NewReader(p.In)
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}

	v := strings.TrimSpace(strings.ToLower(line))
	switch v {
	case "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	case "":
		return defaultYes, nil
	default:
		return defaultYes, nil
	}
}
