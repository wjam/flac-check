package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goyek/goyek/v3"
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

		var stderr strings.Builder
		ext := strings.TrimSpace(stdout.String())
		if !cmd.Exec(a,
			fmt.Sprintf(`go build -trimpath -ldflags="-dumpdep -s -w" -o bin/flac-check%s .`, ext),
			cmd.Env("CGO_ENABLED", "0"),
			cmd.Stderr(&stderr),
		) {
			return
		}

		err := os.WriteFile(filepath.Join("bin", "deps.txt"), []byte(stderr.String()), 0600)
		if err != nil {
			a.Fatal(err)
		}
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
