package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/asraa/slsa-on-github/build-go/pkg"
)

func usage(p string) {
	panic(fmt.Sprintf(`Usage: 
	 %s build [--dry] slsa-releaser.yml
	 %s provenance --binary-name $NAME --digest $DIGEST`, p, p))
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

var (
	// Boolean to indicate a dry run.
	dry bool
)

func main() {
	// Build command
	buildCmd := flag.NewFlagSet("build", flag.ExitOnError)
	buildDry := buildCmd.Bool("dry", false, "dry run of the build without invoking compiler")

	// Provenance command
	provenanceCmd := flag.NewFlagSet("provenance", flag.ExitOnError)
	provenanceName := provenanceCmd.String("binary-name", "", "untrusted binary name of the artifact built")
	provenanceDigest := provenanceCmd.String("digest", "", "sha256 digest of the untrusted binary")

	// Expect a sub-command.
	if len(os.Args) < 2 {
		usage(os.Args[0])
	}

	switch os.Args[1] {
	case buildCmd.Name():
		buildCmd.Parse(os.Args[2:])
		if len(buildCmd.Args()) < 1 {
			usage(os.Args[0])
		}

		goc, err := exec.LookPath("go")
		check(err)

		cfg, err := pkg.ConfigFromFile(buildCmd.Args()[0])
		check(err)
		fmt.Println(cfg)

		gobuild := pkg.GoBuildNew(goc, cfg)

		// Set env variables encoded as arguments.
		err = gobuild.SetArgEnvVariables(buildCmd.Args()[1])
		check(err)

		err = gobuild.Run(*buildDry)
		check(err)
	case provenanceCmd.Name():
		provenanceCmd.Parse(os.Args[2:])
		if *provenanceName == "" || *provenanceDigest == "" {
			usage(os.Args[0])
		}

		githubContext, ok := os.LookupEnv("GITHUB_CONTEXT")
		if !ok {
			panic(errors.New("Environment variable GITHUB_CONTEXT not present"))
		}

		attBytes, err := pkg.GenerateProvenance(*provenanceName, *provenanceDigest, githubContext)
		check(err)

		filename := fmt.Sprintf("%s.intoto.sig", *provenanceName)
		err = ioutil.WriteFile(filename, attBytes, 0600)
		check(err)

		fmt.Printf("::set-output name=signed-provenance-name::%s\n", filename)
	default:
		fmt.Println("expected 'build' or 'provenance' subcommands")
		os.Exit(1)
	}
}
