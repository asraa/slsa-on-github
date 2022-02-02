package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/asraa/slsa-on-github/pkg/provenance"
	"github.com/google/go-github/v40/github"
	"golang.org/x/oauth2"
)

func main() {
	// get org and repo flags
	repoStr := flag.String("repository", "", "owner and repository to fetch build logs, e.g. ossf/scorecard")
	digestStr := flag.String("digest", "", "sha256 digest of the input binary being produced")
	flag.Parse()

	if *repoStr == "" {
		flag.Usage()
		return
	}

	if *digestStr == "" {
		flag.Usage()
		return
	}

	digest := *digestStr
	if _, err := hex.DecodeString(digest); err != nil && len(digest) != 64 {
		fmt.Fprintln(os.Stderr, "sha256 digest is not valid")
		os.Exit(1)
	}

	// split string
	z := strings.SplitN(*repoStr, "/", 2)
	if z == nil || len(z) != 2 {
		flag.Usage()
		return
	}
	org := z[0]
	repo := z[1]

	// make github client
	token, ok := os.LookupEnv("GITHUB_TOKEN")
	if !ok {
		fmt.Printf("%s", "missing GITHUB_TOKEN")
	}
	ctx := context.Background()
	// Requires a token with repo scope
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	workflow, run, job, err := provenance.GetCurrentWorkflowRunAndBuildJob(ctx, client, org, repo)
	if err != nil {
		log.Fatal(err)
	}

	att, err := provenance.GenerateAttestation(workflow, run, job, digest)
	if err != nil {
		log.Fatal(err)
	}
	attBytes, err := json.MarshalIndent(att, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(attBytes))
}
