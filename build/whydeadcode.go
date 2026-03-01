package main

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/goyek/goyek/v3"
	"github.com/goyek/x/cmd"
)

var whyDeadCode = goyek.Define(goyek.Task{
	Name:  "deadcode",
	Usage: "verify Go still using deadcode elimination",
	Action: func(a *goyek.A) {
		deps, err := os.ReadFile(filepath.Join("bin", "deps.txt"))
		if err != nil {
			a.Fatal(err)
		}
		cmd.Exec(a, "go tool -modfile=./tools/whydeadcode/go.mod whydeadcode -fail", cmd.Stdin(bytes.NewBuffer(deps)))

	},
	Deps: []*goyek.DefinedTask{goBuild},
})
