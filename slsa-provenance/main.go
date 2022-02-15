package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	cjson "github.com/docker/go/canonical/json"
	intoto "github.com/in-toto/in-toto-golang/in_toto"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	dsselib "github.com/secure-systems-lab/go-securesystemslib/dsse"
	"github.com/sigstore/cosign/cmd/cosign/cli/fulcio"
	"github.com/sigstore/cosign/pkg/providers"
	_ "github.com/sigstore/cosign/pkg/providers/all"
	"github.com/sigstore/sigstore/pkg/signature/dsse"
)

const (
	defaultFulcioAddr   = "https://fulcio.sigstore.dev"
	defaultOIDCIssuer   = "https://oauth2.sigstore.dev/auth"
	defaultOIDCClientID = "sigstore"
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
		panic(errors.New("Environment variable DIGEST not present"))
	}

	binary, ok := os.LookupEnv("UNTRUSTED_BINARY_NAME")
	if !ok {
		panic(errors.New("Environment variable UNTRUSTED_BINARY_NAME not present"))
	}

	githubContext, ok := os.LookupEnv("GITHUB_CONTEXT")
	if !ok {
		panic(errors.New("Environment variable GITHUB_CONTEXT not present"))
	}

	gh := &GitHubContext{}
	if err := json.Unmarshal([]byte(githubContext), gh); err != nil {
		panic(err)
	}

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

	// Get Fulcio signer
	ctx := context.Background()
	if !providers.Enabled(ctx) {
		panic(fmt.Errorf("no auth provider for fulcio is enabled"))
	}

	fClient, err := fulcio.NewClient(defaultFulcioAddr)
	if err != nil {
		panic(err)
	}
	tok, err := providers.Provide(ctx, defaultOIDCClientID)
	if err != nil {
		panic(err)
	}
	k, err := fulcio.NewSigner(ctx, tok, defaultOIDCIssuer, defaultOIDCClientID, "", fClient)
	if err != nil {
		panic(err)
	}
	wrappedSigner := dsse.WrapSigner(k, intoto.PayloadType)

	attBytes, err := cjson.MarshalCanonical(att)
	if err != nil {
		panic(err)
	}

	signedAtt, err := wrappedSigner.SignMessage(bytes.NewReader(attBytes))
	if err != nil {
		panic(err)
	}

	envelope := &dsselib.Envelope{}
	if err = json.Unmarshal(signedAtt, envelope); err != nil {
		panic(err)
	}

	payload, err := json.MarshalIndent(envelope, "", "\t")
	if err != nil {
		panic(err)
	}

	fmt.Printf(string(payload))
	fmt.Printf(`::set-output name=provenance::%s`, string(payload))
}
