package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/goyek/goyek/v2"
	"github.com/goyek/x/cmd"
)

var goGenerate = goyek.Define(goyek.Task{
	Name:  "go-generate",
	Usage: "go generate",
	Action: func(a *goyek.A) {
		cmd.Exec(a, "go generate ./...")
	},
})

var goTest = goyek.Define(goyek.Task{
	Name:  "go-test",
	Usage: "go test",
	Action: func(a *goyek.A) {
		out := filepath.Join("bin", "coverage.out")
		html := filepath.Join("bin", "coverage.html")
		var pkgs strings.Builder
		if !cmd.Exec(a, "go list -f '{{ .ImportPath }}' ./...", cmd.Stdout(&pkgs)) {
			return
		}
		packages := strings.Join(append(strings.Split(strings.TrimSpace(pkgs.String()), "\n"), "."), ",")
		if !cmd.Exec(a,
			fmt.Sprintf("go test -race -covermode=atomic -coverprofile=%q -coverpkg=%q ./...", out, packages),
		) {
			return
		}
		cmd.Exec(a, fmt.Sprintf("go tool cover -html=%q -o %q", out, html))
	},
	Deps: []*goyek.DefinedTask{mkdirBin},
})

var goBuild = goyek.Define(goyek.Task{
	Name:  "go-build",
	Usage: "go build",
	Action: func(a *goyek.A) {
		var stdout strings.Builder
		if !cmd.Exec(a, "go env GOEXE", cmd.Stdout(&stdout)) {
			return
		}

		ext := strings.TrimSpace(stdout.String())
		cmd.Exec(a,
			fmt.Sprintf(`go build -trimpath -ldflags="-s -w" -o bin/flac-check%s .`, ext),
			cmd.Env("CGO_ENABLED", "0"),
		)
	},
	Deps: []*goyek.DefinedTask{mkdirBin},
})

var goModTidyDiff = goyek.Define(goyek.Task{
	Name:  "go-mod-tidy",
	Usage: "go mod tidy",
	Action: func(a *goyek.A) {
		cmd.Exec(a, "go mod tidy -diff")
	},
})
