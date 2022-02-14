package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/asraa/slsa-on-github/build-go/pkg"
)

func usage(p string) {
	panic(fmt.Sprintf("Usage: %s <config.yml> <binary-name> <env1:val1,env2:val2>\n", p))
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	if len(os.Args) <= 3 {
		usage(os.Args[0])
	}
	goc, err := exec.LookPath("go")
	check(err)

	cfg, err := pkg.ConfigFromFile(os.Args[1])
	check(err)
	fmt.Println(cfg)

	gobuild := pkg.GoBuildNew(goc, cfg)

	// Set output name.
	err = gobuild.SetOutputFilename(os.Args[2])
	check(err)

	// Set env variables encoded as arguments.
	err = gobuild.SetArgEnvVariables(os.Args[3])
	check(err)

	err = gobuild.Run()
	check(err)
}
