package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/asraa/slsa-on-github/build-go/pkg"
)

func usage(p string) {
	panic(fmt.Sprintf("Usage: %s <flags> <config.yml> <env1:val1,env2:val2>\n", p))
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	vendor := flag.Bool("vendor", false, "vendor dependencies")
	flag.Parse()

	if len(flag.Args()) < 1 {
		usage(os.Args[0])
	}

	goc, err := exec.LookPath("go")
	check(err)

	cfg, err := pkg.ConfigFromFile(flag.Args()[0])
	check(err)
	fmt.Println(cfg)

	gobuild := pkg.GoBuildNew(goc, cfg)

	// Set env variables encoded as arguments.
	err = gobuild.SetArgEnvVariables(flag.Args()[1:])
	check(err)

	if *vendor {
		err = gobuild.Vendor(flag.Args()[0])
	} else {
		err = gobuild.Run()
	}

	check(err)
}
