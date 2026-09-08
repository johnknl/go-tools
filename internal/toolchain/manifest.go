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

// Package toolchain defines pinned tools and default values.
package toolchain

// ToolSpec describes one pinned external tool.
type ToolSpec struct {
	Name    string
	Package string
	Version string
}

// InstallTarget returns package@version for go get -tool.
func (t ToolSpec) InstallTarget() string {
	return t.Package + "@" + t.Version
}

// ToolSpecs is the pinned manifest used by tool resolution.
var ToolSpecs = []ToolSpec{
	{Name: "svu", Package: "github.com/caarlos0/svu/v3", Version: "v3.4.1"},
	{Name: "gremlins", Package: "github.com/go-gremlins/gremlins/cmd/gremlins", Version: "v0.6.0"},
	{Name: "golangci-lint", Package: "github.com/golangci/golangci-lint/v2/cmd/golangci-lint", Version: "v2.12.2"},
	{Name: "addlicense", Package: "github.com/google/addlicense", Version: "v1.2.0"},
	{Name: "docsync", Package: "github.com/johnknl/alog/tools/cmd/docsync", Version: DefaultAlogToolsVersion},
	{Name: "mockery", Package: "github.com/vektra/mockery/v2", Version: "v2.53.6"},
	{Name: "gorelease", Package: "golang.org/x/exp/cmd/gorelease", Version: "v0.0.0-20260611194520-c48552f49976"},
	{Name: "benchstat", Package: "golang.org/x/perf/cmd/benchstat", Version: "v0.0.0-20260709024250-82a0b07e230d"},
	{Name: "govulncheck", Package: "golang.org/x/vuln/cmd/govulncheck", Version: "v1.5.0"},
	{
		Name:    "fieldalignment",
		Package: "golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment",
		Version: "v0.44.0",
	},
}

// FindToolSpec locates a tool by binary name.
func FindToolSpec(name string) (ToolSpec, bool) {
	for _, spec := range ToolSpecs {
		if spec.Name == name {
			return spec, true
		}
	}

	return ToolSpec{}, false
}
