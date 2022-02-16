package pkg

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func errCmp(e1, e2 error) bool {
	return errors.Is(e1, e2) || errors.Is(e2, e1)
}

func TestTopLevelEnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected error
	}{
		{
			name:     "no top level env",
			path:     "./testdata/workflow-no-top-env.yml",
			expected: nil,
		},
		{
			name:     "top level env but not var defined",
			path:     "./testdata/workflow-top-env-novar.yml",
			expected: nil,
		},
		{
			name:     "top level env single empty var",
			path:     "./testdata/workflow-top-env-emptyvalue.yml",
			expected: errorTopLevelEnvVariables,
		},
		{
			name:     "top level env two empty var",
			path:     "./testdata/workflow-top-env-twoemptyvalue.yml",
			expected: errorTopLevelEnvVariables,
		},
		{
			name:     "top level env one empty one set",
			path:     "./testdata/workflow-top-env-one-set.yml",
			expected: errorTopLevelEnvVariables,
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
			workflow, err := WorkflowFromBytes(content)
			if err != nil {
				panic(fmt.Errorf("WorkflowFromBytes: %w", err))
			}

			err = workflow.validateTopLevelEnvironmentVariables()
			if !errCmp(err, tt.expected) {
				t.Errorf(cmp.Diff(err, tt.expected))
			}
		})
	}
}
