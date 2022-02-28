package pkg

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func errCmp(e1, e2 error) bool {
	return errors.Is(e1, e2) || errors.Is(e2, e1)
}

func TestConfigFromFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected error
	}{
		{
			name:     "valid releaser",
			path:     "./testdata/releaser-valid.yml",
			expected: nil,
		},
		{
			name:     "missing version",
			path:     "./testdata/releaser-noversion.yml",
			expected: errorUnsupportedVersion,
		},
		{
			name:     "invalid version",
			path:     "./testdata/releaser-invalid-version.yml",
			expected: errorUnsupportedVersion,
		},
		{
			name:     "invalid envs",
			path:     "./testdata/releaser-invalid-envs.yml",
			expected: errorInvalidEnvironmentVariable,
		},
	}
	for _, tt := range tests {
		tt := tt // Re-initializing variable so it is not changed while executing the closure below
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := ConfigFromFile(tt.path)
			if !errCmp(err, tt.expected) {
				t.Errorf(cmp.Diff(err, tt.expected))
			}
		})
	}
}
