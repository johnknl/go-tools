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

// Package testcmd implements test command workflows.
package testcmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/johnknl/go-tools/internal/commands/docscmd"
	"github.com/johnknl/go-tools/internal/commands/lintcmd"
	"github.com/johnknl/go-tools/internal/runtime/appctx"
	"github.com/johnknl/go-tools/internal/runtime/execx"
	"github.com/johnknl/go-tools/internal/runtime/gitx"
	"github.com/johnknl/go-tools/internal/toolchain"
	"github.com/johnknl/go-tools/internal/workflows"
	"github.com/spf13/cobra"
)

const (
	benchSuffixDirty = "-dirty"
	dirPerm          = 0o750
	filePerm         = 0o600
	fuzzListPattern  = "^Fuzz"

	cmdUnit = "unit"
	cmdRun  = "run"
	cmdList = "list"

	pprofTopFlag  = "-top"
	pprofListFlag = "-list"

	benchAuto = "auto"
	headRef   = "HEAD"
	prevRef   = "HEAD~1"
)

type options struct {
	Yes     bool
	NoInput bool
}

type benchOptions struct {
	storeDir string
	benchDir string
	pattern  string
	ref      string
	a        string
	b        string
	fileTag  string
	count    int
}

// NewCommand builds the test root command.
func NewCommand(app *appctx.Context) *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Run test workflows",
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			app.Yes = opts.Yes
			app.NoInput = opts.NoInput
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return Execute(app, []string{cmdUnit})
		},
	}
	cmd.PersistentFlags().BoolVar(&opts.Yes, "yes", false, "assume yes for prompts")
	cmd.PersistentFlags().BoolVar(&opts.NoInput, "no-input", false, "disable interactive prompts")

	cmd.AddCommand(newUnitCommand(app))
	cmd.AddCommand(newCoverageCommand(app))
	cmd.AddCommand(newAllCommand(app))
	cmd.AddCommand(newBenchCommand(app))
	cmd.AddCommand(newFuzzCommand(app))
	cmd.AddCommand(newMutCommand(app))
	cmd.AddCommand(newProfCommand(app))
	cmd.AddCommand(newMocksCommand(app))
	cmd.AddCommand(newCICommand(app))

	return cmd
}

// Execute runs the test command tree.
func Execute(app *appctx.Context, args []string) error {
	cmd := NewCommand(app)
	cmd.SetArgs(args)
	return cmd.Execute()
}

func newUnitCommand(app *appctx.Context) *cobra.Command {
	return &cobra.Command{
		Use:   cmdUnit,
		Short: "Run untagged tests",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runGo(cmd.Context(), app, goTestArgs("-timeout=1m", "-count=10", "-race", "-shuffle=on", "./...")...)
		},
	}
}

func newCoverageCommand(app *appctx.Context) *cobra.Command {
	coverageFile := toolchain.DefaultCoverageFile(app.CWD)

	return &cobra.Command{
		Use:   "coverage",
		Short: "Run coverage suite",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := os.MkdirAll(filepath.Dir(coverageFile), dirPerm); err != nil {
				return err
			}

			pkgs, err := coveragePackages(cmd.Context(), app)
			if err != nil {
				return err
			}

			args := goTestArgs("-v", "-race", "-covermode=atomic", "-coverprofile="+coverageFile)
			args = append(args, pkgs...)
			if err := runGo(cmd.Context(), app, args...); err != nil {
				return err
			}

			return runGo(cmd.Context(), app, "tool", "cover", "-func="+coverageFile)
		},
	}
}

func newAllCommand(app *appctx.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "all",
		Short: "Run full test suites",
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := Execute(app, []string{cmdUnit}); err != nil {
				return err
			}
			if err := Execute(app, []string{"fuzz", cmdRun}); err != nil {
				return err
			}
			if err := Execute(app, []string{"mut", cmdRun}); err != nil {
				return err
			}
			return Execute(app, []string{"bench", cmdRun})
		},
	}
}

func newBenchCommand(app *appctx.Context) *cobra.Command {
	bo := &benchOptions{
		benchDir: toolchain.DefaultBenchDir(app.CWD),
		storeDir: toolchain.DefaultBenchStoreDir(app.CWD),
		pattern:  toolchain.DefaultBenchPattern,
		count:    toolchain.DefaultBenchCount,
		ref:      toolchain.DefaultBenchRef,
		a:        toolchain.DefaultBenchA,
		b:        toolchain.DefaultBenchB,
	}

	cmd := &cobra.Command{Use: "bench", Short: "Benchmark workflows"}
	runCmd := &cobra.Command{
		Use:   cmdRun,
		Short: "Run benchmark snapshot",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runBenchSnapshot(cmd.Context(), app, bo)
		},
	}
	compareCmd := &cobra.Command{
		Use:   "compare",
		Short: "Compare benchmark snapshots",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return compareBenchSnapshots(cmd.Context(), app, bo)
		},
	}
	listCmd := &cobra.Command{
		Use:   cmdList,
		Short: "List snapshots",
		RunE: func(_ *cobra.Command, _ []string) error {
			return listBenchSnapshots(app, bo)
		},
	}

	for _, sub := range []*cobra.Command{runCmd, compareCmd, listCmd} {
		sub.Flags().StringVar(&bo.pattern, "pattern", bo.pattern, "benchmark pattern")
		sub.Flags().StringVar(&bo.fileTag, "file-tag", bo.fileTag, "snapshot file tag")
		sub.Flags().StringVar(&bo.storeDir, "store-dir", bo.storeDir, "snapshot storage directory")
		sub.Flags().StringVar(&bo.benchDir, "bench-dir", bo.benchDir, "benchmark temp directory")
	}
	runCmd.Flags().IntVar(&bo.count, "count", bo.count, "benchmark count")
	runCmd.Flags().StringVar(&bo.ref, "ref", bo.ref, "benchmark git ref")
	compareCmd.Flags().StringVar(&bo.a, "a", bo.a, "baseline ref or auto")
	compareCmd.Flags().StringVar(&bo.b, "b", bo.b, "comparison ref or auto")

	cmd.AddCommand(runCmd, compareCmd, listCmd)
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runCmd.RunE(cmd, nil)
	}

	return cmd
}

func newFuzzCommand(app *appctx.Context) *cobra.Command {
	count := toolchain.DefaultFuzzCount
	fuzzTime := toolchain.DefaultFuzzTime
	pkg := toolchain.DefaultFuzzPkg
	target := toolchain.DefaultFuzzTarget
	parallel := toolchain.DefaultFuzzParallel
	goMaxProcs := toolchain.DefaultFuzzGOMAXPROCS
	goMemLimit := toolchain.DefaultFuzzGOMEMLIMIT
	goGC := toolchain.DefaultFuzzGOGC

	vmName := toolchain.DefaultVMName
	vmBaseImage := toolchain.DefaultVMBaseImage
	vmCPUs := toolchain.DefaultVMCPUs
	vmMemoryMiB := toolchain.DefaultVMMemoryMiB
	vmDiskGiB := toolchain.DefaultVMDiskGiB
	vmWorkdir := toolchain.DefaultVMWorkdir
	vmStorageDir := toolchain.DefaultVMStorageDir(vmName)
	libvirtURI := toolchain.DefaultLibvirtURI
	vmFuzzTime := toolchain.DefaultVMFuzzTime
	vmFuzzCount := toolchain.DefaultVMFuzzCount
	vmFuzzTarget := toolchain.DefaultVMFuzzTarget
	vmFuzzParallel := toolchain.DefaultVMFuzzParallel
	vmFuzzGOMAXPROCS := toolchain.DefaultVMFuzzGOMAXPROCS
	vmFuzzGOMEMLIMIT := toolchain.DefaultVMFuzzGOMEMLIMIT
	vmFuzzGOGC := toolchain.DefaultVMFuzzGOGC
	fuzzVMScript := filepath.Join("make", "scripts", "fuzz-vm.sh")

	cmd := &cobra.Command{Use: "fuzz", Short: "Fuzz workflows"}
	runCmd := &cobra.Command{
		Use:   cmdRun,
		Short: "Run fuzz tests",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			env := []string{"GOMAXPROCS=" + goMaxProcs, "GOMEMLIMIT=" + goMemLimit, "GOGC=" + goGC}

			if target != "." {
				pkgs, err := goList(ctx, app, pkg)
				if err != nil {
					return err
				}
				if len(pkgs) != 1 {
					return fmt.Errorf("--target requires --pkg to resolve to exactly one package")
				}
				return runFuzzTarget(ctx, app, env, count, parallel, target, fuzzTime, pkgs[0])
			}

			pkgs, err := goList(ctx, app, pkg)
			if err != nil {
				return err
			}

			for _, p := range pkgs {
				targets, listErr := listFuzzTargets(ctx, app, env, p)
				if listErr != nil {
					return listErr
				}
				if len(targets) == 0 {
					_, _ = fmt.Fprintf(app.Stdout, "==> %s: no fuzz targets\n", p)
					continue
				}
				for _, fuzzTarget := range targets {
					_, _ = fmt.Fprintf(app.Stdout, "Running: %s:%s\n", p, fuzzTarget)
					expr := "^" + fuzzTarget + "$"
					if err := runFuzzTarget(ctx, app, env, count, parallel, expr, fuzzTime, p); err != nil {
						return err
					}
				}
			}

			return nil
		},
	}

	sandboxCmd := func(name string, action string) *cobra.Command {
		return &cobra.Command{
			Use:   name,
			Short: "Run fuzz sandbox " + action,
			RunE: func(cmd *cobra.Command, _ []string) error {
				env := []string{
					"VM_NAME=" + vmName,
					"VM_BASE_IMAGE=" + vmBaseImage,
					fmt.Sprintf("VM_CPUS=%d", vmCPUs),
					fmt.Sprintf("VM_MEMORY_MIB=%d", vmMemoryMiB),
					fmt.Sprintf("VM_DISK_GIB=%d", vmDiskGiB),
					"VM_WORKDIR=" + vmWorkdir,
					"VM_STORAGE_DIR=" + vmStorageDir,
					"LIBVIRT_URI=" + libvirtURI,
					fmt.Sprintf("FUZZ_COUNT=%d", vmFuzzCount),
					"FUZZ_TIME=" + vmFuzzTime,
					"FUZZ_TARGET=" + vmFuzzTarget,
					fmt.Sprintf("FUZZ_PARALLEL=%d", vmFuzzParallel),
					"FUZZ_GOMAXPROCS=" + vmFuzzGOMAXPROCS,
					"FUZZ_GOMEMLIMIT=" + vmFuzzGOMEMLIMIT,
					"FUZZ_GOGC=" + vmFuzzGOGC,
				}
				opts := execx.RunOptions{
					Dir:    app.CWD,
					Env:    env,
					Stdout: app.Stdout,
					Stderr: app.Stderr,
					Stdin:  app.Stdin,
				}
				return app.Runner.Run(cmd.Context(), fuzzVMScript, []string{action}, opts)
			},
		}
	}

	attachVMFlags := func(c *cobra.Command) {
		c.Flags().StringVar(&fuzzVMScript, "sandbox-script", fuzzVMScript, "path to fuzz VM helper script")
		c.Flags().StringVar(&vmName, "vm-name", vmName, "VM name")
		c.Flags().StringVar(&vmBaseImage, "vm-base-image", vmBaseImage, "VM base image")
		c.Flags().IntVar(&vmCPUs, "vm-cpus", vmCPUs, "VM CPUs")
		c.Flags().IntVar(&vmMemoryMiB, "vm-memory-mib", vmMemoryMiB, "VM memory MiB")
		c.Flags().IntVar(&vmDiskGiB, "vm-disk-gib", vmDiskGiB, "VM disk GiB")
		c.Flags().StringVar(&vmWorkdir, "vm-workdir", vmWorkdir, "VM workdir")
		c.Flags().StringVar(&vmStorageDir, "vm-storage-dir", vmStorageDir, "VM storage directory")
		c.Flags().StringVar(&libvirtURI, "libvirt-uri", libvirtURI, "libvirt URI")
		c.Flags().IntVar(&vmFuzzCount, "vm-fuzz-count", vmFuzzCount, "sandbox fuzz count")
		c.Flags().StringVar(&vmFuzzTime, "vm-fuzz-time", vmFuzzTime, "sandbox fuzz time")
		c.Flags().StringVar(&vmFuzzTarget, "vm-fuzz-target", vmFuzzTarget, "sandbox fuzz target")
		c.Flags().IntVar(&vmFuzzParallel, "vm-fuzz-parallel", vmFuzzParallel, "sandbox fuzz parallel")
		c.Flags().StringVar(&vmFuzzGOMAXPROCS, "vm-fuzz-gomaxprocs", vmFuzzGOMAXPROCS, "sandbox GOMAXPROCS")
		c.Flags().StringVar(&vmFuzzGOMEMLIMIT, "vm-fuzz-gomemlimit", vmFuzzGOMEMLIMIT, "sandbox GOMEMLIMIT")
		c.Flags().StringVar(&vmFuzzGOGC, "vm-fuzz-gogc", vmFuzzGOGC, "sandbox GOGC")
	}

	runCmd.Flags().IntVar(&count, "count", count, "fuzz count")
	runCmd.Flags().StringVar(&fuzzTime, "time", fuzzTime, "fuzz time")
	runCmd.Flags().StringVar(&pkg, "pkg", pkg, "package pattern")
	runCmd.Flags().StringVar(&target, "target", target, "fuzz target")
	runCmd.Flags().IntVar(&parallel, "parallel", parallel, "parallel runs")
	runCmd.Flags().StringVar(&goMaxProcs, "gomaxprocs", goMaxProcs, "GOMAXPROCS")
	runCmd.Flags().StringVar(&goMemLimit, "gomemlimit", goMemLimit, "GOMEMLIMIT")
	runCmd.Flags().StringVar(&goGC, "gogc", goGC, "GOGC")

	provisionCmd := sandboxCmd("sandbox-provision", "provision")
	runSandboxCmd := sandboxCmd("sandbox-run", "run")
	sshSandboxCmd := sandboxCmd("sandbox-ssh", "ssh")
	stopSandboxCmd := sandboxCmd("sandbox-stop", "stop")
	destroySandboxCmd := sandboxCmd("sandbox-destroy", "destroy")
	for _, c := range []*cobra.Command{
		provisionCmd,
		runSandboxCmd,
		sshSandboxCmd,
		stopSandboxCmd,
		destroySandboxCmd,
	} {
		attachVMFlags(c)
	}

	cmd.AddCommand(runCmd, provisionCmd, runSandboxCmd, sshSandboxCmd, stopSandboxCmd, destroySandboxCmd)
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runCmd.RunE(cmd, nil)
	}

	return cmd
}

func newMutCommand(app *appctx.Context) *cobra.Command {
	mutBin := ""
	configPath := ".gremlins.yaml"
	resultsDir := toolchain.DefaultMutResultsDir(app.CWD)
	resultsFile := toolchain.DefaultMutResultsFile(app.CWD)
	workers := toolchain.DefaultMutWorkers
	testCPU := toolchain.DefaultMutTestCPU
	timeoutCoefficient := toolchain.DefaultMutTimeoutCoefficient
	baseBranch := toolchain.DefaultMutBaseBranch

	cmd := &cobra.Command{Use: "mut", Short: "Mutation testing"}

	commonArgs := func() []string {
		return []string{
			"--config", configPath,
			"--coverpkg", "./...",
			"--workers", fmt.Sprintf("%d", workers),
			"--test-cpu", fmt.Sprintf("%d", testCPU),
			"--timeout-coefficient", fmt.Sprintf("%d", timeoutCoefficient),
			"--exclude-files", "^internal/storage/mocks/",
			"--exclude-files", "^tools/cmd/docsync/",
			"--output", resultsFile,
		}
	}

	runGremlins := func(ctx context.Context, args []string) error {
		if err := workflows.EnsureGremlinsConfig(app, configPath); err != nil {
			return err
		}
		if err := os.MkdirAll(resultsDir, dirPerm); err != nil {
			return err
		}
		if mutBin != "" {
			opts := execx.RunOptions{Dir: app.CWD, Stdout: app.Stdout, Stderr: app.Stderr}
			return app.Runner.Run(ctx, mutBin, args, opts)
		}
		return app.RunTool(ctx, "gremlins", args)
	}

	runCmd := &cobra.Command{
		Use:   cmdRun,
		Short: "Run mutation tests",
		RunE: func(cmd *cobra.Command, _ []string) error {
			args := append([]string{"unleash", ".", "--integration"}, commonArgs()...)
			return runGremlins(cmd.Context(), args)
		},
	}
	dryRunCmd := &cobra.Command{
		Use:   "dry-run",
		Short: "Dry-run mutation analysis",
		RunE: func(cmd *cobra.Command, _ []string) error {
			args := append([]string{"unleash", ".", "--dry-run"}, commonArgs()...)
			return runGremlins(cmd.Context(), args)
		},
	}
	runDiffCmd := &cobra.Command{
		Use:   "run-diff",
		Short: "Run mutation tests on diff",
		RunE: func(cmd *cobra.Command, _ []string) error {
			args := make([]string, 0, 5+len(commonArgs()))
			args = append(args, "unleash", ".", "--integration", "--diff", baseBranch)
			args = append(args, commonArgs()...)
			return runGremlins(cmd.Context(), args)
		},
	}

	for _, c := range []*cobra.Command{runCmd, dryRunCmd, runDiffCmd} {
		c.Flags().StringVar(&mutBin, "bin", mutBin, "gremlins binary path")
		c.Flags().StringVar(&configPath, "config", configPath, "gremlins config file")
		c.Flags().StringVar(&resultsDir, "results-dir", resultsDir, "results directory")
		c.Flags().StringVar(&resultsFile, "results-file", resultsFile, "results file")
		c.Flags().IntVar(&workers, "workers", workers, "worker count")
		c.Flags().IntVar(&testCPU, "test-cpu", testCPU, "test CPU")
		c.Flags().IntVar(&timeoutCoefficient, "timeout-coefficient", timeoutCoefficient, "timeout coefficient")
	}
	runDiffCmd.Flags().StringVar(&baseBranch, "base-branch", baseBranch, "base branch")

	cmd.AddCommand(runCmd, dryRunCmd, runDiffCmd)
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runCmd.RunE(cmd, nil)
	}

	return cmd
}

func newProfCommand(app *appctx.Context) *cobra.Command {
	benchDir := toolchain.DefaultBenchDir(app.CWD)
	benchPattern := toolchain.DefaultProfBenchPattern
	benchCount := toolchain.DefaultProfBenchCount
	benchPkg := toolchain.DefaultProfBenchPkg
	pprofListFunc := toolchain.DefaultPPROFListFunc

	cmd := &cobra.Command{Use: "prof", Short: "Profiling workflows"}

	profileFiles := func() (string, string, string, string) {
		stem := profileStem(benchPattern)
		cpu := filepath.Join(benchDir, fmt.Sprintf("cpu.%s.%d.pprof", stem, benchCount))
		mem := filepath.Join(benchDir, fmt.Sprintf("mem.%s.%d.pprof", stem, benchCount))
		block := filepath.Join(benchDir, fmt.Sprintf("block.%s.%d.pprof", stem, benchCount))
		mutex := filepath.Join(benchDir, fmt.Sprintf("mutex.%s.%d.pprof", stem, benchCount))
		return cpu, mem, block, mutex
	}

	runBenchWith := func(ctx context.Context, extra []string) error {
		if err := os.MkdirAll(benchDir, dirPerm); err != nil {
			return err
		}
		args := goTestArgs("-run", "^$", "-bench", benchPattern, "-benchmem", "-count", fmt.Sprintf("%d", benchCount))
		args = append(args, extra...)
		args = append(args, benchPkg)
		return runGo(ctx, app, args...)
	}

	profileCmd := &cobra.Command{
		Use:   "profile",
		Short: "Capture CPU and memory profiles",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cpu, mem, _, _ := profileFiles()
			return runBenchWith(cmd.Context(), []string{"-cpuprofile", cpu, "-memprofile", mem})
		},
	}
	profileLocksCmd := &cobra.Command{
		Use:   "profile-locks",
		Short: "Capture block and mutex profiles",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, _, block, mutex := profileFiles()
			return runBenchWith(cmd.Context(), []string{"-blockprofile", block, "-mutexprofile", mutex})
		},
	}

	topMemCmd := pprofReadCommand(
		app,
		"top-mem",
		"Show top allocators",
		profileCmd,
		profileFiles,
		"mem",
		[]string{pprofTopFlag},
		pprofListFunc,
	)
	topCPUCmd := pprofReadCommand(
		app,
		"top-cpu",
		"Show top CPU functions",
		profileCmd,
		profileFiles,
		"cpu",
		[]string{pprofTopFlag},
		pprofListFunc,
	)
	topBlockCmd := pprofReadCommand(
		app,
		"top-block",
		"Show top blocking functions",
		profileLocksCmd,
		profileFiles,
		"block",
		[]string{pprofTopFlag},
		pprofListFunc,
	)
	topMutexCmd := pprofReadCommand(
		app,
		"top-mutex",
		"Show top mutex contention",
		profileLocksCmd,
		profileFiles,
		"mutex",
		[]string{pprofTopFlag},
		pprofListFunc,
	)
	listMemCmd := pprofReadCommand(
		app,
		"list-mem",
		"List alloc lines",
		profileCmd,
		profileFiles,
		"mem",
		[]string{pprofListFlag},
		pprofListFunc,
	)
	listCPUCmd := pprofReadCommand(
		app,
		"list-cpu",
		"List CPU lines",
		profileCmd,
		profileFiles,
		"cpu",
		[]string{pprofListFlag},
		pprofListFunc,
	)

	escapeCmd := &cobra.Command{
		Use:   "escape",
		Short: "Run escape analysis",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runGo(cmd.Context(), app, goTestArgs("-run", "^$", "-bench", benchPattern, "-gcflags=all=-m=2", benchPkg)...)
		},
	}

	for _, c := range []*cobra.Command{
		profileCmd,
		profileLocksCmd,
		topMemCmd,
		topCPUCmd,
		topBlockCmd,
		topMutexCmd,
		listMemCmd,
		listCPUCmd,
		escapeCmd,
	} {
		c.Flags().StringVar(&benchDir, "bench-dir", benchDir, "profile output directory")
		c.Flags().StringVar(&benchPattern, "pattern", benchPattern, "benchmark pattern")
		c.Flags().IntVar(&benchCount, "count", benchCount, "benchmark count")
		c.Flags().StringVar(&benchPkg, "pkg", benchPkg, "benchmark package")
		c.Flags().StringVar(&pprofListFunc, "list-func", pprofListFunc, "pprof list function")
	}

	cmd.AddCommand(
		profileCmd,
		profileLocksCmd,
		topMemCmd,
		topCPUCmd,
		topBlockCmd,
		topMutexCmd,
		listMemCmd,
		listCPUCmd,
		escapeCmd,
	)
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return profileCmd.RunE(cmd, nil)
	}

	return cmd
}

func newMocksCommand(app *appctx.Context) *cobra.Command {
	mockeryBin := ""

	return &cobra.Command{
		Use:   "mocks",
		Short: "Generate mocks",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if mockeryBin != "" {
				opts := execx.RunOptions{Dir: app.CWD, Stdout: app.Stdout, Stderr: app.Stderr}
				return app.Runner.Run(cmd.Context(), mockeryBin, nil, opts)
			}
			return app.RunTool(cmd.Context(), "mockery", nil)
		},
	}
}

func newCICommand(app *appctx.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "ci",
		Short: "Run CI workflow",
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := Execute(app, []string{"coverage"}); err != nil {
				return err
			}
			if err := lintcmd.Execute(app, []string{"run", "--yes"}); err != nil {
				return err
			}
			if err := lintcmd.Execute(app, []string{"vuln"}); err != nil {
				return err
			}
			return docscmd.Execute(app, []string{"check"})
		},
	}
}

//nolint:gocyclo // mirrors make parity flow for bench snapshot behavior.
func runBenchSnapshot(ctx context.Context, app *appctx.Context, bo *benchOptions) error {
	if err := os.MkdirAll(bo.storeDir, dirPerm); err != nil {
		return err
	}

	resolved, err := gitx.ResolveCommit(ctx, app.Runner, app.CWD, bo.ref)
	if err != nil {
		return err
	}
	short, err := gitx.ShortCommit(ctx, app.Runner, app.CWD, resolved, toolchain.FixedBenchSHALen)
	if err != nil {
		return err
	}

	tag := benchTag(bo.pattern, bo.fileTag)
	tagSuffix := ""
	if tag != "" {
		tagSuffix = "--" + tag
	}

	runDir := app.CWD
	suffix := ""
	head, err := gitx.HeadCommit(ctx, app.Runner, app.CWD)
	if err != nil {
		return err
	}
	if head == resolved {
		dirty, dirtyErr := gitx.IsDirty(ctx, app.Runner, app.CWD)
		if dirtyErr != nil {
			return dirtyErr
		}
		if dirty {
			suffix = benchSuffixDirty
		}
	} else {
		refWorktree := filepath.Join(bo.benchDir, ".ref-worktree")
		_ = os.RemoveAll(refWorktree)
		worktreeAddArgs := []string{"worktree", "add", "-f", "--detach", refWorktree, resolved}
		worktreeAddOpts := execx.RunOptions{Dir: app.CWD, Stdout: app.Stdout, Stderr: app.Stderr}
		if runErr := app.Runner.Run(ctx, "git", worktreeAddArgs, worktreeAddOpts); runErr != nil {
			return err
		}
		//nolint:contextcheck // cleanup should continue if parent context is canceled.
		defer func() {
			removeArgs := []string{"worktree", "remove", "--force", refWorktree}
			_ = app.Runner.Run(context.Background(), "git", removeArgs, execx.RunOptions{Dir: app.CWD})
		}()
		runDir = refWorktree
	}

	outFile := filepath.Join(bo.storeDir, short+suffix+tagSuffix+".txt")
	tmpFile := outFile + ".tmp"
	openFlags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC

	//nolint:gosec // user-provided output path for local snapshot file.
	outHandle, err := os.OpenFile(tmpFile, openFlags, filePerm)
	if err != nil {
		return err
	}
	defer func() {
		_ = outHandle.Close()
	}()

	goArgs := make([]string, 0, 10)
	goArgs = append(goArgs, "-C", runDir)
	goArgs = append(
		goArgs,
		goTestArgs(
			"-run",
			"^$",
			"-bench",
			bo.pattern,
			"-benchmem",
			fmt.Sprintf("-count=%d", bo.count),
			"./...",
		)...,
	)
	goOpts := execx.RunOptions{
		Dir:    app.CWD,
		Env:    []string{"GOWORK=off"},
		Stdout: io.MultiWriter(outHandle, app.Stdout),
		Stderr: app.Stderr,
	}
	if err := app.Runner.Run(ctx, "go", goArgs, goOpts); err != nil {
		return err
	}

	if err := os.Rename(tmpFile, outFile); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(app.Stdout, "stored benchmark snapshot: %s\n", outFile)

	return nil
}

func compareBenchSnapshots(ctx context.Context, app *appctx.Context, bo *benchOptions) error {
	tag := benchTag(bo.pattern, bo.fileTag)
	tagSuffix := ""
	if tag != "" {
		tagSuffix = "--" + tag
	}

	head, err := gitx.HeadCommit(ctx, app.Runner, app.CWD)
	if err != nil {
		return err
	}
	dirty, err := gitx.IsDirty(ctx, app.Runner, app.CWD)
	if err != nil {
		return err
	}

	modeDirty := false
	aRef := ""
	bRef := ""
	if bo.a == benchAuto && bo.b == benchAuto {
		if dirty {
			aRef = head
			bRef = head
			modeDirty = true
		} else {
			aRef, err = gitx.ResolveCommit(ctx, app.Runner, app.CWD, prevRef)
			if err != nil {
				return err
			}
			bRef = head
		}
	} else {
		aName := bo.a
		bName := bo.b
		if aName == benchAuto {
			aName = prevRef
		}
		if bName == benchAuto {
			bName = headRef
		}
		aRef, err = gitx.ResolveCommit(ctx, app.Runner, app.CWD, aName)
		if err != nil {
			return err
		}
		bRef, err = gitx.ResolveCommit(ctx, app.Runner, app.CWD, bName)
		if err != nil {
			return err
		}
		if bName == headRef && dirty && aRef == head {
			modeDirty = true
		}
	}

	aFile, err := benchFileFor(ctx, app, bo.storeDir, aRef, false, tagSuffix)
	if err != nil {
		return err
	}
	bFile, err := benchFileFor(ctx, app, bo.storeDir, bRef, modeDirty, tagSuffix)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(app.Stdout, "comparing %s\n", aFile)
	_, _ = fmt.Fprintf(app.Stdout, "      with %s\n", bFile)
	return app.RunTool(ctx, "benchstat", []string{aFile, bFile})
}

func listBenchSnapshots(app *appctx.Context, bo *benchOptions) error {
	if err := os.MkdirAll(bo.storeDir, dirPerm); err != nil {
		return err
	}
	entries, err := os.ReadDir(bo.storeDir)
	if err != nil {
		return err
	}

	files := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".txt") {
			continue
		}
		files = append(files, filepath.Join(bo.storeDir, e.Name()))
	}
	sort.Strings(files)
	for _, file := range files {
		_, _ = fmt.Fprintln(app.Stdout, file)
	}

	return nil
}

func runGo(ctx context.Context, app *appctx.Context, args ...string) error {
	opts := execx.RunOptions{Dir: app.CWD, Stdout: app.Stdout, Stderr: app.Stderr}
	return app.Runner.Run(ctx, "go", args, opts)
}

func goTestArgs(args ...string) []string {
	res := make([]string, 1, 1+len(args))
	res[0] = "test"
	res = append(res, args...)
	return res
}

func listFuzzTargets(ctx context.Context, app *appctx.Context, env []string, pkg string) ([]string, error) {
	args := goTestArgs("-list", fuzzListPattern, pkg)
	opts := execx.RunOptions{Dir: app.CWD, Env: env}
	out, err := app.Runner.Output(ctx, "go", args, opts)
	if err != nil {
		return nil, err
	}

	targets := make([]string, 0)
	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Fuzz") {
			targets = append(targets, line)
		}
	}

	return targets, nil
}

func runFuzzTarget(
	ctx context.Context,
	app *appctx.Context,
	env []string,
	count int,
	parallel int,
	target string,
	fuzzTime string,
	pkg string,
) error {
	args := goTestArgs(
		"-timeout=2m",
		fmt.Sprintf("-count=%d", count),
		fmt.Sprintf("-parallel=%d", parallel),
		"-run=^$",
		"-fuzz",
		target,
		"-fuzztime",
		fuzzTime,
		pkg,
	)
	opts := execx.RunOptions{Dir: app.CWD, Env: env, Stdout: app.Stdout, Stderr: app.Stderr}
	return app.Runner.Run(ctx, "go", args, opts)
}

func pprofReadCommand(
	app *appctx.Context,
	use string,
	short string,
	gen *cobra.Command,
	profiles func() (string, string, string, string),
	kind string,
	operation []string,
	listFunc string,
) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cpu, mem, block, mutex := profiles()
			path := mem
			switch kind {
			case "cpu":
				path = cpu
			case "block":
				path = block
			case "mutex":
				path = mutex
			}

			if err := ensureFile(path, gen); err != nil {
				return err
			}

			args := []string{"tool", "pprof"}
			if len(operation) > 0 {
				if operation[0] == pprofListFlag {
					args = append(args, "-list="+listFunc)
				} else {
					args = append(args, operation...)
				}
			}
			args = append(args, path)
			return runGo(cmd.Context(), app, args...)
		},
	}
}

func goList(ctx context.Context, app *appctx.Context, pkg string) ([]string, error) {
	out, err := app.Runner.Output(ctx, "go", []string{"list", pkg}, execx.RunOptions{Dir: app.CWD})
	if err != nil {
		return nil, err
	}

	res := make([]string, 0)
	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			res = append(res, line)
		}
	}

	return res, nil
}

func coveragePackages(ctx context.Context, app *appctx.Context) ([]string, error) {
	out, err := app.Runner.Output(ctx, "go", []string{"list", "./..."}, execx.RunOptions{Dir: app.CWD})
	if err != nil {
		return nil, err
	}

	pkgs := make([]string, 0)
	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasSuffix(line, "/internal/storage/mocks") {
			continue
		}
		pkgs = append(pkgs, line)
	}

	return pkgs, nil
}

func benchTag(pattern string, explicit string) string {
	if explicit != "" {
		return explicit
	}
	if pattern == "." {
		return ""
	}
	re := regexp.MustCompile(`[^[:alnum:]._-]+`)
	tag := re.ReplaceAllString(pattern, "_")
	tag = strings.Trim(tag, "_")
	if tag == "" {
		return "pattern"
	}

	return tag
}

func benchFileFor(
	ctx context.Context,
	app *appctx.Context,
	storeDir string,
	ref string,
	dirty bool,
	tagSuffix string,
) (string, error) {
	short, err := gitx.ShortCommit(ctx, app.Runner, app.CWD, ref, toolchain.FixedBenchSHALen)
	if err != nil {
		return "", err
	}

	name := short
	if dirty {
		name += benchSuffixDirty
	}
	name += tagSuffix + ".txt"
	path := filepath.Join(storeDir, name)

	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("missing snapshot: %s", path)
		}
		return "", err
	}

	return path, nil
}

func profileStem(pattern string) string {
	re := regexp.MustCompile(`[^[:alnum:]]`)
	return re.ReplaceAllString(pattern, "_")
}

func ensureFile(file string, generator *cobra.Command) error {
	if _, err := os.Stat(file); err == nil {
		return nil
	}
	if err := generator.RunE(generator, nil); err != nil {
		return err
	}
	_, err := os.Stat(file)
	if err != nil {
		return fmt.Errorf("expected profile %s: %w", file, err)
	}

	return nil
}
