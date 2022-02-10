package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	// Check for org and repo env variables
	repository, ok := os.LookupEnv("GITHUB_REPOSITORY")
	if !ok {
		log.Fatal(errors.New("Environment variable GITHUB_REPOSITORY not present"))
	}
	digest, ok := os.LookupEnv("DIGEST")
	if !ok {
		log.Fatal(errors.New("Environment variable DIGEST not present"))
	}

	binary, ok := os.LookupEnv("UNTRUSTED_BINARY_NAME")
	if !ok {
		log.Fatal(errors.New("Environment variable UNTRUSTED_BINARY_NAME not present"))
	}

	fmt.Println("binary", binary)
	// // Check for GITHUB env variables
	// ghRunIdStr, ok := os.LookupEnv("GITHUB_RUN_ID")
	// if !ok {
	// 	fmt.Fprintln(os.Stderr, "Environment variable GITHUB_RUN_ID not present")
	// 	os.Exit(1)
	// }

	// ghRunId, err := strconv.ParseInt(ghRunIdStr, 10, 64)
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "Invalid github run ID string: %v", err)
	// 	os.Exit(1)
	// }

	if _, err := hex.DecodeString(digest); err != nil || len(digest) != 64 {
		log.Fatal(fmt.Errorf("sha256 digest is not valid: %s", digest))
	}

	// split string
	z := strings.SplitN(repository, "/", 2)
	if z == nil || len(z) != 2 {
		log.Fatal(errors.New("sha256 digest is not valid"))
	}
	// org := z[0]
	// repo := z[1]

	// // make github client
	// token, ok := os.LookupEnv("GITHUB_TOKEN")
	// if !ok {
	// 	log.Fatal(errors.New("missing GITHUB_TOKEN"))
	// }
	// ctx := context.Background()
	// // Requires a token with repo scope
	// ts := oauth2.StaticTokenSource(
	// 	&oauth2.Token{AccessToken: token},
	// )
	// tc := oauth2.NewClient(ctx, ts)
	// client := github.NewClient(tc)

	// workflow, run, job, err := provenance.GetCurrentWorkflowRunAndBuildJob(ctx, client, org, repo, ghRunId)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// att, err := provenance.GenerateAttestation(workflow, run, job, digest)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	att := "{my-into-attestation}"
	attBytes, err := json.Marshal(att)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf(`::set-output name=provenance::%s`, string(attBytes))
}
