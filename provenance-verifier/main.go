package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/asraa/slsa-on-github/provenance-verifier/pkg"
	"github.com/sigstore/cosign/cmd/cosign/cli/rekor"
)

func usage(p string) {
	panic(fmt.Sprintf("Usage: %s TODO\n", p))
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

var (
	provenancePath string
	binaryPath     string
)

var (
	defaultRekorAddr = "https://rekor.sigstore.dev"
)

func verify(ctx context.Context, provenancePath string, artifactHash string) error {
	provenance, err := os.ReadFile(provenancePath)
	if err != nil {
		return fmt.Errorf("os.ReadFile: %w", err)
	}

	env, err := pkg.EnvelopeFromBytes(provenance)
	if err != nil {
		return err
	}

	rClient, err := rekor.NewClient(defaultRekorAddr)
	if err != nil {
		return err
	}

	// Get Rekor entries corresponding to the binary artifact (matching Subject digest) in the provenance.
	uuids, err := pkg.GetRekorEntries(rClient, *env, artifactHash)
	if err != nil {
		return err
	}

	// Verify the provenance and return the signing certificate.
	cert, err := pkg.FindSigningCertificate(ctx, uuids, *env, rClient)
	if err != nil {
		return err
	}

	// Get the workflow info given the certificate information.
	workflowInfo, err := pkg.GetWorkflowInfoFromCertificate(cert)
	if err != nil {
		return err
	}

	if err := pkg.VerifyWorkflowIdentity(workflowInfo); err != nil {
		return err
	}

	b, err := json.MarshalIndent(workflowInfo, "", "\t")
	if err != nil {
		return err
	}

	fmt.Printf("verified SLSA provenance produced at \n %s\n", b)
	return nil
}

func main() {
	flag.StringVar(&provenancePath, "provenance", "", "path to a provenance file")
	flag.StringVar(&binaryPath, "binary", "", "path to a binary to verify")
	flag.Parse()

	if provenancePath == "" || binaryPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	f, err := os.Open(binaryPath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	if err := verify(ctx, provenancePath, hex.EncodeToString(h.Sum(nil))); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	fmt.Println("successfully verified SLSA provenance")
}
