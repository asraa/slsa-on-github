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
			expected: errorDeclaredEnv,
		},
		{
			name:     "top level env two empty var",
			path:     "./testdata/workflow-top-env-twoemptyvalue.yml",
			expected: errorDeclaredEnv,
		},
		{
			name:     "top level env one empty one set",
			path:     "./testdata/workflow-top-env-one-set.yml",
			expected: errorDeclaredEnv,
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

			err = workflow.validateTopLevelEnv()
			if !errCmp(err, tt.expected) {
				t.Errorf(cmp.Diff(err, tt.expected))
			}
		})
	}
}

func TestJobLevelEnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected map[string]error
	}{
		{
			name: "no job level env",
			path: "./testdata/workflow-no-job-env.yml",
			expected: map[string]error{
				"args": nil, "build": nil, "upload": nil,
			},
		},
		{
			name: "job level env defined but empty",
			path: "./testdata/workflow-job-env-empty.yml",
			expected: map[string]error{
				"args": nil, "build": nil, "upload": nil,
			},
		},
		{
			name: "job level env defined one set single empty",
			path: "./testdata/workflow-job-env-one-set-single.yml",
			expected: map[string]error{
				"args":   errorDeclaredEnv,
				"build":  errorDeclaredEnv,
				"upload": errorDeclaredEnv,
			},
		},
		{
			name: "job level env defined and set",
			path: "./testdata/workflow-job-env-set.yml",
			expected: map[string]error{
				"args":   errorDeclaredEnv,
				"build":  errorDeclaredEnv,
				"upload": errorDeclaredEnv,
			},
		},
		{
			name: "job level env mix",
			path: "./testdata/workflow-job-env-mix.yml",
			expected: map[string]error{
				"args":   errorDeclaredEnv,
				"job2":   errorDeclaredEnv,
				"job3":   errorDeclaredEnv,
				"job4":   nil,
				"build":  nil,
				"upload": errorDeclaredEnv,
				"job5":   nil,
				"job6":   errorDeclaredEnv,
			},
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

			if len(workflow.workflow.Jobs) == 0 {
				panic(fmt.Errorf("no jobs in the workflow: %s", tt.name))
			}
			for name, job := range workflow.workflow.Jobs {
				val, exists := tt.expected[name]
				if !exists {
					panic(fmt.Errorf("%s job does not exist", name))
				}
				err = workflow.validateJobLevelEnv(job)
				if !errCmp(err, val) {
					t.Errorf(cmp.Diff(err, val))
				}
			}
		})
	}
}
