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

import "path/filepath"

const (
	// FixedBenchSHALen matches the benchmark snapshot filename hash width.
	FixedBenchSHALen = 12
	// DefaultBuildTags are applied to formatting and lint-related Go commands.
	DefaultBuildTags = "tools"
	// DefaultDocsDir is the default documentation directory.
	DefaultDocsDir = "docs"
	// DefaultBenchPattern mirrors the former Makefile benchmark pattern default.
	DefaultBenchPattern = "."
	// DefaultBenchCount mirrors the former Makefile benchmark count default.
	DefaultBenchCount = 6
	// DefaultBenchRef mirrors the former Makefile benchmark reference default.
	DefaultBenchRef = "HEAD"
	// DefaultBenchA mirrors the former Makefile benchmark compare A ref default.
	DefaultBenchA = "auto"
	// DefaultBenchB mirrors the former Makefile benchmark compare B ref default.
	DefaultBenchB = "auto"
	// DefaultFuzzCount mirrors the former Makefile fuzz count default.
	DefaultFuzzCount = 1
	// DefaultFuzzTime mirrors the former Makefile fuzz time default.
	DefaultFuzzTime = "20s"
	// DefaultFuzzPkg mirrors the former Makefile fuzz package default.
	DefaultFuzzPkg = "./..."
	// DefaultFuzzTarget mirrors the former Makefile fuzz target default.
	DefaultFuzzTarget = "."
	// DefaultFuzzParallel mirrors the former Makefile fuzz parallel default.
	DefaultFuzzParallel = 1
	// DefaultFuzzGOMAXPROCS mirrors the former Makefile fuzz GOMAXPROCS default.
	DefaultFuzzGOMAXPROCS = "1"
	// DefaultFuzzGOMEMLIMIT mirrors the former Makefile fuzz GOMEMLIMIT default.
	DefaultFuzzGOMEMLIMIT = "1024MiB"
	// DefaultFuzzGOGC mirrors the former Makefile fuzz GOGC default.
	DefaultFuzzGOGC = "50"

	// DefaultVMName mirrors the former Makefile VM name default.
	DefaultVMName = "alog-fuzz"
	// DefaultVMBaseImage mirrors the former Makefile VM base image default.
	DefaultVMBaseImage = "https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img"
	// DefaultVMCPUs mirrors the former Makefile VM CPU default.
	DefaultVMCPUs = 16
	// DefaultVMMemoryMiB mirrors the former Makefile VM memory default.
	DefaultVMMemoryMiB = 32768
	// DefaultVMDiskGiB mirrors the former Makefile VM disk size default.
	DefaultVMDiskGiB = 32
	// DefaultVMWorkdir mirrors the former Makefile VM workdir default.
	DefaultVMWorkdir = "/home/alog/alog"
	// DefaultLibvirtURI mirrors the former Makefile libvirt URI default.
	DefaultLibvirtURI = "qemu:///system"

	// DefaultVMFuzzTime mirrors the former Makefile VM fuzz time default.
	DefaultVMFuzzTime = "90s"
	// DefaultVMFuzzCount mirrors the former Makefile VM fuzz count default.
	DefaultVMFuzzCount = 1
	// DefaultVMFuzzTarget mirrors the former Makefile VM fuzz target default.
	DefaultVMFuzzTarget = "."
	// DefaultVMFuzzParallel mirrors the former Makefile VM fuzz parallel default.
	DefaultVMFuzzParallel = 1
	// DefaultVMFuzzGOMAXPROCS mirrors the former Makefile VM fuzz GOMAXPROCS default.
	DefaultVMFuzzGOMAXPROCS = "1"
	// DefaultVMFuzzGOMEMLIMIT mirrors the former Makefile VM fuzz GOMEMLIMIT default.
	DefaultVMFuzzGOMEMLIMIT = "1536MiB"
	// DefaultVMFuzzGOGC mirrors the former Makefile VM fuzz GOGC default.
	DefaultVMFuzzGOGC = "100"

	// DefaultMutWorkers mirrors the former Makefile mutation worker default.
	DefaultMutWorkers = 2
	// DefaultMutTestCPU mirrors the former Makefile mutation test CPU default.
	DefaultMutTestCPU = 1
	// DefaultMutTimeoutCoefficient mirrors the former Makefile timeout coefficient default.
	DefaultMutTimeoutCoefficient = 4
	// DefaultMutBaseBranch mirrors the former Makefile mutation base branch default.
	DefaultMutBaseBranch = "main"

	// DefaultProfBenchPattern mirrors the former Makefile profile benchmark pattern default.
	DefaultProfBenchPattern = "^BenchmarkArchive/append$"
	// DefaultProfBenchCount mirrors the former Makefile profile benchmark count default.
	DefaultProfBenchCount = 10
	// DefaultProfBenchPkg mirrors the former Makefile profile benchmark package default.
	DefaultProfBenchPkg = "."
	// DefaultPPROFListFunc mirrors the former Makefile pprof list function default.
	DefaultPPROFListFunc = "github.com/johnknl/alog.(*Archive).Range"

	// DefaultAlogToolsVersion pins docsync from the alog tools module.
	DefaultAlogToolsVersion = "v0.0.0-20260905162208-522cb3a7a00a"
)

// DefaultBenchDir returns the default benchmark scratch directory.
func DefaultBenchDir(cwd string) string { return filepath.Join(cwd, ".tmp", "bench") }

// DefaultBenchStoreDir returns the default benchmark snapshot directory.
func DefaultBenchStoreDir(cwd string) string { return filepath.Join(cwd, "bench", "results") }

// DefaultCoverageFile returns the default coverage profile path.
func DefaultCoverageFile(cwd string) string { return filepath.Join(cwd, ".tmp", "coverage.out") }

// DefaultMutResultsDir returns the default mutation results directory.
func DefaultMutResultsDir(cwd string) string { return filepath.Join(cwd, ".tmp", "mutation") }

// DefaultMutResultsFile returns the default mutation results file path.
func DefaultMutResultsFile(cwd string) string {
	return filepath.Join(DefaultMutResultsDir(cwd), "gremlins.json")
}

// DefaultVMStorageDir returns the default VM storage path for a VM name.
func DefaultVMStorageDir(vmName string) string {
	return filepath.Join("/var/lib/libvirt/images", vmName)
}
