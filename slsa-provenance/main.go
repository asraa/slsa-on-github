package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	intoto "github.com/in-toto/in-toto-golang/in_toto"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
)

type GitHubContext struct {
	Repository string `json:"repository"`
	ActionPath string `json:"action_path"`
	Workflow   string `json:"workflow"`
	RunId      string `json:"run_id"`
	EventName  string `json:"event_name"`
	SHA        string `json:"sha"`
	Token      string `json:"token,omitempty"`
	RunNumber  string `json:"run_number"`
}

func main() {
	digest, ok := os.LookupEnv("DIGEST")
	if !ok {
		log.Fatal(errors.New("Environment variable DIGEST not present"))
	}

	binary, ok := os.LookupEnv("UNTRUSTED_BINARY_NAME")
	if !ok {
		log.Fatal(errors.New("Environment variable UNTRUSTED_BINARY_NAME not present"))
	}

	githubContext, ok := os.LookupEnv("GITHUB_CONTEXT")
	if !ok {
		log.Fatal(errors.New("Environment variable GITHUB_CONTEXT not present"))
	}

	gh := &GitHubContext{}
	if err := json.Unmarshal([]byte(githubContext), gh); err != nil {
		log.Fatal(err)
	}

	fmt.Println("binary", binary)
	fmt.Println(gh.Repository)
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

	att := intoto.ProvenanceStatement{
		StatementHeader: intoto.StatementHeader{
			Type:          intoto.StatementInTotoV01,
			PredicateType: slsa.PredicateSLSAProvenance,
			Subject: []intoto.Subject{
				{
					Name: binary,
					Digest: slsa.DigestSet{
						"sha256": digest,
					},
				},
			},
		},
		Predicate: slsa.ProvenancePredicate{
			BuildType: "https://github.com/Attestations/GitHubActionsWorkflow@v1",
			Builder: slsa.ProvenanceBuilder{
				ID: "https://github.com/Attestations/GitHubHostedActions@v1",
			},
			Invocation: slsa.ProvenanceInvocation{
				ConfigSource: slsa.ConfigSource{
					EntryPoint: gh.Workflow,
					URI:        fmt.Sprintf("git+%s.git", gh.Repository),
					Digest: slsa.DigestSet{
						"SHA1": gh.SHA,
					},
				},
				// Add event inputs
				Environment: map[string]interface{}{
					"arch": "amd64", // TODO: Does GitHub run actually expose this?
					"env": map[string]string{
						"GITHUB_RUN_NUMBER": gh.RunNumber,
						"GITHUB_RUN_ID":     gh.RunId,
						"GITHUB_EVENT_NAME": gh.EventName,
					},
				},
			},
			Materials: []slsa.ProvenanceMaterial{{
				URI: fmt.Sprintf("git+%s.git", gh.Repository),
				Digest: slsa.DigestSet{
					"SHA1": gh.SHA,
				}},
			},
		},
	}

	// att, err := provenance.GenerateAttestation(workflow, run, job, digest)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	attBytes, err := json.Marshal(att)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf(string(attBytes))
	fmt.Printf(`::set-output name=provenance::%s`, string(attBytes))
}
