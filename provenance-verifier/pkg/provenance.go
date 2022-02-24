package pkg

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	cjson "github.com/docker/go/canonical/json"
	"github.com/go-openapi/runtime"
	"github.com/google/go-github/v39/github"
	"github.com/google/trillian/merkle/logverifier"
	"github.com/google/trillian/merkle/rfc6962"
	intoto "github.com/in-toto/in-toto-golang/in_toto"
	dsselib "github.com/secure-systems-lab/go-securesystemslib/dsse"
	"github.com/sigstore/sigstore/pkg/signature/dsse"

	"github.com/sigstore/cosign/cmd/cosign/cli/fulcio"
	"github.com/sigstore/cosign/pkg/cosign"
	"github.com/sigstore/cosign/pkg/cosign/bundle"
	sigs "github.com/sigstore/cosign/pkg/signature"
	"github.com/sigstore/rekor/pkg/generated/client"
	"github.com/sigstore/rekor/pkg/generated/client/entries"
	"github.com/sigstore/rekor/pkg/generated/client/index"
	"github.com/sigstore/rekor/pkg/generated/models"
	"github.com/sigstore/rekor/pkg/types"
	intotod "github.com/sigstore/rekor/pkg/types/intoto/v0.0.1"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
)

const (
	defaultFulcioAddr   = "https://fulcio.sigstore.dev"
	defaultOIDCIssuer   = "https://oauth2.sigstore.dev/auth"
	defaultOIDCClientID = "sigstore"
	defaultRekorAddr    = "https://rekor.sigstore.dev"
	certOidcIssuer      = "https://token.actions.githubusercontent.com"
)

var (
	errorInvalidDssePayload = errors.New("invalid DSSE envelope payload")
	errorRekorSearch        = errors.New("error searching rekor entries")
)

// Get SHA256 Subject Digest from the provenance statement.
func getSha256Digest(env dsselib.Envelope) (string, error) {
	pyld, err := base64.StdEncoding.DecodeString(env.Payload)
	if err != nil {
		return "", fmt.Errorf("%w: %s", errorInvalidDssePayload, "decoding payload")
	}
	prov := &intoto.ProvenanceStatement{}
	if err := json.Unmarshal([]byte(pyld), prov); err != nil {
		return "", fmt.Errorf("%w: %s", errorInvalidDssePayload, "unmarshalling json")
	}
	if len(prov.Subject) == 0 {
		return "", fmt.Errorf("%w: %s", errorInvalidDssePayload, "no subjects")
	}
	digestSet := prov.Subject[0].Digest
	hash, exists := digestSet["sha256"]
	if !exists {
		return "", fmt.Errorf("%w: %s", errorInvalidDssePayload, "no sha256 subject digest")
	}
	return hash, nil
}

// GetRekorEntries finds all entry UUIDs by the digest of the artifact binary.
func GetRekorEntries(rClient *client.Rekor, env dsselib.Envelope) ([]string, error) {
	// Get Subject Digest from the provenance statement.
	hash, err := getSha256Digest(env)
	if err != nil {
		return nil, err
	}

	// Use search index to find rekor entry UUIDs that match Subject Digest.
	params := index.NewSearchIndexParams()
	params.Query = &models.SearchIndex{Hash: fmt.Sprintf("sha256%v", hash)}
	resp, err := rClient.Index.SearchIndex(params)
	if err != nil {
		return nil, errorRekorSearch
	}

	if len(resp.Payload) == 0 {
		return nil, fmt.Errorf("%w: no matching entries found", errorRekorSearch)
	}

	return resp.GetPayload(), nil
}

func verifyTlogEntry(ctx context.Context, rekorClient *client.Rekor, uuid string) (*models.LogEntryAnon, error) {
	params := entries.NewGetLogEntryByUUIDParamsWithContext(ctx)
	params.EntryUUID = uuid

	lep, err := rekorClient.Entries.GetLogEntryByUUID(params)
	if err != nil {
		return nil, err
	}

	if len(lep.Payload) != 1 {
		return nil, errors.New("UUID value can not be extracted")
	}
	e := lep.Payload[params.EntryUUID]
	if e.Verification == nil || e.Verification.InclusionProof == nil {
		return nil, errors.New("inclusion proof not provided")
	}

	hashes := [][]byte{}
	for _, h := range e.Verification.InclusionProof.Hashes {
		hb, _ := hex.DecodeString(h)
		hashes = append(hashes, hb)
	}

	rootHash, _ := hex.DecodeString(*e.Verification.InclusionProof.RootHash)
	leafHash, _ := hex.DecodeString(params.EntryUUID)

	v := logverifier.New(rfc6962.DefaultHasher)
	if err := v.VerifyInclusionProof(*e.Verification.InclusionProof.LogIndex, *e.Verification.InclusionProof.TreeSize, hashes, rootHash, leafHash); err != nil {
		return nil, fmt.Errorf("%w: %s", err, "verifying inclusion proof")
	}

	// Verify rekor's signature over the SET.
	payload := bundle.RekorPayload{
		Body:           e.Body,
		IntegratedTime: *e.IntegratedTime,
		LogIndex:       *e.LogIndex,
		LogID:          *e.LogID,
	}

	pub, err := cosign.GetRekorPub(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, "unable to fetch Rekor public keys from TUF repository")
	}
	rekorPubKey, err := cosign.PemToECDSAKey(pub)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, "rekor pem to ecdsa")
	}
	err = cosign.VerifySET(payload, []byte(e.Verification.SignedEntryTimestamp), rekorPubKey)
	return &e, err
}

func extractCert(e *models.LogEntryAnon) (*x509.Certificate, error) {
	b, err := base64.StdEncoding.DecodeString(e.Body.(string))
	if err != nil {
		return nil, err
	}

	pe, err := models.UnmarshalProposedEntry(bytes.NewReader(b), runtime.JSONConsumer())
	if err != nil {
		return nil, err
	}

	eimpl, err := types.NewEntry(pe)
	if err != nil {
		return nil, err
	}

	var publicKeyB64 []byte
	switch e := eimpl.(type) {
	case *intotod.V001Entry:
		publicKeyB64, err = e.IntotoObj.PublicKey.MarshalText()
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unexpected tlog entry type")
	}

	publicKey, err := base64.StdEncoding.DecodeString(string(publicKeyB64))
	if err != nil {
		return nil, err
	}

	certs, err := cryptoutils.UnmarshalCertificatesFromPEM(publicKey)
	if err != nil {
		return nil, err
	}

	if len(certs) != 1 {
		return nil, errors.New("unexpected number of cert pem tlog entry")
	}

	return certs[0], err
}

// FindSigningCertificate finds and verifies a matching signing certificate from a list of Rekor entry UUIDs.
func FindSigningCertificate(ctx context.Context, uuids []string, dssePayload dsselib.Envelope, rClient *client.Rekor) (*x509.Certificate, error) {
	attBytes, err := cjson.MarshalCanonical(dssePayload)
	if err != nil {
		return nil, err
	}
	// Iterate through each matching UUID and perform:
	//   * Verify TLOG entry (inclusion and signed entry timestamp against Rekor pubkey).
	//   * Verify the signing certificate against the Fulcio root CA.
	//   * Verify dsse envelope signature against signing certificate.
	//   * Check signature expiration against IntegratedTime in entry.
	//   * If all succeed, return the signing certificate.
	for _, uuid := range uuids {
		entry, err := verifyTlogEntry(ctx, rClient, uuid)
		if err != nil {
			continue
		}
		cert, err := extractCert(entry)
		if err != nil {
			continue
		}
		co := &cosign.CheckOpts{
			RootCerts:      fulcio.GetRoots(),
			CertOidcIssuer: certOidcIssuer,
		}
		verifier, err := cosign.ValidateAndUnpackCert(cert, co)
		if err != nil {
			continue
		}
		verifier = dsse.WrapVerifier(verifier)
		if err := verifier.VerifySignature(bytes.NewReader(attBytes), bytes.NewReader(attBytes)); err != nil {
			continue
		}
		it := time.Unix(*entry.IntegratedTime, 0)
		if err := cosign.CheckExpiry(cert, it); err != nil {
			continue
		}
		// success!
		return cert, nil
	}

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
