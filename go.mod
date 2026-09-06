module github.com/johnknl/go-tools

go 1.26.7

require github.com/spf13/cobra v1.10.2

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
)

tool (
	github.com/johnknl/go-tools/cmd/bench
	github.com/johnknl/go-tools/cmd/docs
	github.com/johnknl/go-tools/cmd/fuzz
	github.com/johnknl/go-tools/cmd/lint
	github.com/johnknl/go-tools/cmd/mut
	github.com/johnknl/go-tools/cmd/prof
	github.com/johnknl/go-tools/cmd/release
	github.com/johnknl/go-tools/cmd/test
	github.com/johnknl/go-tools/cmd/tools
)
