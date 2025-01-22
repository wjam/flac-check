package main

import "github.com/goyek/goyek/v2"

var stageInit = goyek.Define(goyek.Task{
	Name:  "init",
	Usage: "init stage",
	Deps: goyek.Deps{
		goGenerate,
		fmtCheck,
	},
})

var stageTest = goyek.Define(goyek.Task{
	Name:  "test",
	Usage: "test stage",
	Deps: goyek.Deps{
		goVet,
		goTest,
	},
})

var stageBuild = goyek.Define(goyek.Task{
	Name:  "build",
	Usage: "build stage",
	Deps: goyek.Deps{
		goBuild,
	},
})
