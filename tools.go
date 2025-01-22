//go:build tools

package main

// TODO don't commit this - wait for Go 1.24 and use golangci-lint instead

import (
	_ "golang.org/x/tools/cmd/goimports"
)
