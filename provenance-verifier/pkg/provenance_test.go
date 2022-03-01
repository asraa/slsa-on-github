package pkg

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/go-openapi/runtime"
	"github.com/google/go-cmp/cmp"
	dsselib "github.com/secure-systems-lab/go-securesystemslib/dsse"
	"github.com/sigstore/rekor/pkg/generated/client"
	"github.com/sigstore/rekor/pkg/generated/client/index"
)

type searchResult struct {
	resp *index.SearchIndexOK
	err  error
}

func envelopeFromBytes(payload []byte) (env *dsselib.Envelope, err error) {
	env = &dsselib.Envelope{}
	err = json.Unmarshal(payload, env)
	return
}

type MockIndexClient struct {
	result searchResult
}

func (m *MockIndexClient) SearchIndex(params *index.SearchIndexParams, opts ...index.ClientOption) (*index.SearchIndexOK, error) {
	return m.result.resp, m.result.err
}

func (m *MockIndexClient) SetTransport(transport runtime.ClientTransport) {
}

func TestGetRekorEntries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		path         string
		artifactHash string
		res          searchResult
		expected     error
	}{
		{
			name:         "invalid dsse: not SLSA predicate",
			path:         "./testdata/dsse-not-slsa.intoto",
			artifactHash: "0ae7e4fa71686538440012ee36a2634dbaa19df2dd16a466f52411fb348bbc4e",
			expected:     errorInvalidDssePayload,
		},
		{
			name:         "invalid dsse: nil subject",
			path:         "./testdata/dsse-no-subject.intoto",
			artifactHash: "0ae7e4fa71686538440012ee36a2634dbaa19df2dd16a466f52411fb348bbc4e",
			expected:     errorInvalidDssePayload,
		},
		{
			name:         "invalid dsse: no sha256 subject digest",
			path:         "./testdata/dsse-no-subject-hash.intoto",
			artifactHash: "0ae7e4fa71686538440012ee36a2634dbaa19df2dd16a466f52411fb348bbc4e",
			expected:     errorInvalidDssePayload,
		},
		{
			name:         "rekor search result error",
			path:         "./testdata/dsse-valid.intoto",
			artifactHash: "0ae7e4fa71686538440012ee36a2634dbaa19df2dd16a466f52411fb348bbc4e",
			res: searchResult{
				err: index.NewSearchIndexDefault(500),
			},
			expected: errorRekorSearch,
		},
		{
			name:         "no rekor entries found",
			path:         "./testdata/dsse-valid.intoto",
			artifactHash: "0ae7e4fa71686538440012ee36a2634dbaa19df2dd16a466f52411fb348bbc4e",
			res: searchResult{
				err: nil,
				resp: &index.SearchIndexOK{
					Payload: []string{},
				},
			},
			expected: errorRekorSearch,
		},
		{
			name:         "mismatched artifact hash with env",
			path:         "./testdata/dsse-valid.intoto",
			artifactHash: "1ae7e4fa71686538440012ee36a2634dbaa19df2dd16a466f52411fb348bbc4e",
			res: searchResult{
				err: nil,
				resp: &index.SearchIndexOK{
					Payload: []string{"39d5109436c43dad92897d50f3b271aa456382875a922b28fedef9038b8f683a"},
				},
			},
			expected: errorMismatchHash,
		},
		{
			name:         "valid rekor entries found",
			path:         "./testdata/dsse-valid.intoto",
			artifactHash: "0ae7e4fa71686538440012ee36a2634dbaa19df2dd16a466f52411fb348bbc4e",
			res: searchResult{
				err: nil,
				resp: &index.SearchIndexOK{
					Payload: []string{"39d5109436c43dad92897d50f3b271aa456382875a922b28fedef9038b8f683a"},
				},
			},
			expected: nil,
		},
	}
	for _, tt := range tests {
		tt := tt // Re-initializing variable so it is not changed while executing the closure below
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			content, err := os.ReadFile(tt.path)
			if err != nil {
				panic(fmt.Errorf("os.ReadFile: %w", err))
			}
			env, err := envelopeFromBytes(content)
			if err != nil {
				panic(fmt.Errorf("envelopeFromBytes: %w", err))
			}

			var mClient client.Rekor
			mClient.Index = &MockIndexClient{result: tt.res}

			_, err = GetRekorEntries(&mClient, *env, tt.artifactHash)
			if !errCmp(err, tt.expected) {
				t.Errorf(cmp.Diff(err, tt.expected))
			}
		})
	}
}
