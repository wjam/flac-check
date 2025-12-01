package main

import (
	"github.com/goyek/goyek/v3"
	"github.com/goyek/x/cmd"
)

var golangciLint = goyek.Define(goyek.Task{
	Name:  "lint",
	Usage: "lint",
	Action: func(a *goyek.A) {
		cmd.Exec(a, "go tool -modfile=./tools/golangci-lint/go.mod golangci-lint run")
	},
})

var _ = goyek.Define(goyek.Task{
	Name:  "lint-fix",
	Usage: "lint-fix",
	Action: func(a *goyek.A) {
		cmd.Exec(a, "go tool -modfile=./tools/golangci-lint/go.mod golangci-lint run --fix")
	},
})
