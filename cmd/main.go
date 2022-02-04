package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/asraa/slsa-on-github/pkg/provenance"
	"github.com/google/go-github/v40/github"
	"golang.org/x/oauth2"
)

func main() {
	// Check for org and repo env variables
	repository, ok := os.LookupEnv("INPUT_REPOSITORY")
	if !ok {
		fmt.Fprintln(os.Stderr, "Environment variable INPUT_REPOSITORY not present")
		os.Exit(1)
	}
	digest, ok := os.LookupEnv("INPUT_DIGEST")
	if !ok {
		fmt.Fprintln(os.Stderr, "Environment variable INPUT_DIGEST not present")
		os.Exit(1)
	}

	// Check for GITHUB env variables
	ghRunIdStr, ok := os.LookupEnv("INPUT_GITHUB_RUN_ID")
	if !ok {
		fmt.Fprintln(os.Stderr, "Environment variable GITHUB_RUN_ID not present")
		os.Exit(1)
	}

	ghRunId, err := strconv.ParseInt(ghRunIdStr, 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid github run ID string: %v", err)
		os.Exit(1)
	}

	if _, err := hex.DecodeString(digest); err != nil && len(digest) != 64 {
		fmt.Fprintln(os.Stderr, "sha256 digest is not valid")
		os.Exit(1)
	}

	// split string
	z := strings.SplitN(repository, "/", 2)
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

	workflow, run, job, err := provenance.GetCurrentWorkflowRunAndBuildJob(ctx, client, org, repo, ghRunId)
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
	fmt.Println(fmt.Sprintf(`::set-output name=provenance::%s`, string(attBytes)))
}
