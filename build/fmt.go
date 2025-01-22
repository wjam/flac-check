package main

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/goyek/goyek/v2"
	"github.com/goyek/x/cmd"
)

// TODO switch to go run golangci-lint when Go 1.24

var _ = goyek.Define(goyek.Task{
	Name:  "fmt",
	Usage: "fmt files",
	Action: func(a *goyek.A) {
		goFiles := find(a, ".go")
		s := strings.Join(goFiles, " ")
		cmd.Exec(a, fmt.Sprintf("go run golang.org/x/tools/cmd/goimports -w %s", s))
	},
})

var fmtCheck = goyek.Define(goyek.Task{
	Name:  "fmtcheck",
	Usage: "fmtcheck files",
	Action: func(a *goyek.A) {
		goFiles := find(a, ".go")
		s := strings.Join(goFiles, " ")

		var stdout strings.Builder
		cmd.Exec(a, fmt.Sprintf("go run golang.org/x/tools/cmd/goimports -l %s", s), cmd.Stdout(&stdout))

		if stdout.Len() == 0 {
			return
		}

		a.Errorf("goimports: %s", stdout.String())
	},
})

func find(a *goyek.A, ext string) []string {
	a.Helper()
	var files []string
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(d.Name()) == ext {
			files = append(files, filepath.ToSlash(path))
		}
		return nil
	})
	if err != nil {
		a.Fatal(err)
	}
	return files
}
