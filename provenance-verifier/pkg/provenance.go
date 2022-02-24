package pkg

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/go-github/v39/github"
	intoto "github.com/in-toto/in-toto-golang/in_toto"
	dsselib "github.com/secure-systems-lab/go-securesystemslib/dsse"
	sigs "github.com/sigstore/cosign/pkg/signature"
	"github.com/sigstore/rekor/pkg/generated/client"
	"github.com/sigstore/rekor/pkg/generated/client/index"
	"github.com/sigstore/rekor/pkg/generated/models"
	"github.com/spf13/viper"
)

const (
	defaultFulcioAddr   = "https://fulcio.sigstore.dev"
	defaultOIDCIssuer   = "https://oauth2.sigstore.dev/auth"
	defaultOIDCClientID = "sigstore"
	defaultRekorAddr    = "https://rekor.sigstore.dev"
)

// GetRekorEntries finds all entry UUIDs by the digest of the artifact binary.
func GetRekorEntries(rClient *client.Rekor, dsse dsselib.Envelope) ([]string, error) {
	// Get Subject Digest from the provenance statement.
	prov := &intoto.ProvenanceStatement{}
	if err := json.Unmarshal([]byte(dsse.Payload), prov); err != nil {
		return nil, err
	}
	if len(prov.Subject) == 0 {
		return nil, errors.New("provenance statement does not contain any subject digests")
	}
	digestSet := prov.Subject[0].Digest
	hash, exists := digestSet["sha256"]
	if !exists {
		return nil, errors.New("digest set does not contain sha256 digest")
	}

	// Use search index to find rekor entry UUIDs that match Subject Digest.
	params := index.NewSearchIndexParams()
	params.Query = &models.SearchIndex{Hash: fmt.Sprintf("sha256%v", hash)}
	resp, err := rClient.Index.SearchIndex(params)
	if err != nil {
		switch t := err.(type) {
		case *index.SearchIndexDefault:
			if t.Code() == http.StatusNotImplemented {
				return nil, fmt.Errorf("search index not enabled on %v", viper.GetString("rekor_server"))
			}
			return nil, err
		default:
			return nil, err
		}
	}

	if len(resp.Payload) == 0 {
		return nil, fmt.Errorf("no matching entries found")
	}

	return resp.GetPayload(), nil
}

// FindSigningCertificate finds and verifies a matching signing certificate from a list of Rekor entry UUIDs.
func FindSigningCertificate(uuids []string, dsse dsselib.Envelope) (*x509.Certificate, error) {
	// Iterate through each matching UUID and perform:
	//   * Verify TLOG entry (inclusion and signed entry timestamp against Rekor pubkey).
	//   * Check if the signing certificate verifies the dsse envelope.
	//   * Verify the signing certificate against the Fulcio root CA.
	//   * Check signature expiration against IntegratedTime in entry.
	//   * If all succeed, return the signing certificate.

	return nil, errors.New("could not find a matching signature entry")
}

func getExtension(cert *x509.Certificate, oid string) string {
	for _, ext := range cert.Extensions {
		if strings.Contains(ext.Id.String(), oid) {
			return string(ext.Value)
		}
	}
	return ""
}

// GetWorkflowFromCertificate gets the workflow content from the Fulcio authenticated content.
func GetWorkflowFromCertificate(cert *x509.Certificate) (*github.RepositoryContent, error) {
	// TODO: Verify trigger?
	jobWorkflowRef := sigs.CertSubject(cert)
	sha := getExtension(cert, "1.3.6.1.4.1.57264.1.3")

	// The job-workflow-ref is `{owner}/{repo}/{path}/{filename}@{ref}`
	jobUri, err := url.Parse(jobWorkflowRef)
	if err != nil {
		return nil, err
	}

	pathParts := strings.SplitN(jobUri.Path, "/", 4)
	org := pathParts[1]
	repo := pathParts[2]
	filePath := strings.SplitN(pathParts[3], "@", 2)

	// Checkout the workflow path at the commit hash from the cert.
	ctx := context.Background()
	client := github.NewClient(nil)
	workflowContent, _, _, err := client.Repositories.GetContents(ctx, org, repo, filePath[0], &github.RepositoryContentGetOptions{Ref: sha})
	return workflowContent, err
}
