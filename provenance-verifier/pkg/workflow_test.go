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

func TestTopLevelDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected error
	}{
		{
			name:     "no top level defaults",
			path:     "./testdata/workflow-no-top-defaults.yml",
			expected: nil,
		},
		{
			name:     "top level defaults",
			path:     "./testdata/workflow-top-defaults.yml",
			expected: errorDeclaredDefaults,
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

			err = workflow.validateTopLevelDefaults()
			if !errCmp(err, tt.expected) {
				t.Errorf(cmp.Diff(err, tt.expected))
			}
		})
	}
}

func TestValidateRunner(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected error
	}{
		{
			name:     "no runner defined",
			path:     "./testdata/workflow-no-runner.yml",
			expected: nil,
		},
		{
			name:     "runners defined GH-hosted",
			path:     "./testdata/workflow-gh-hosted-runners.yml",
			expected: nil,
		},
		{
			name:     "runner self-hosted first",
			path:     "./testdata/workflow-first-self-hosted-runners.yml",
			expected: errorSelfHostedRunner,
		},
		{
			name:     "runner self-hosted second",
			path:     "./testdata/workflow-second-self-hosted-runners.yml",
			expected: errorSelfHostedRunner,
		},
		{
			name:     "runner self-hosted third",
			path:     "./testdata/workflow-third-self-hosted-runners.yml",
			expected: errorSelfHostedRunner,
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

			err = workflow.validateRunner()
			if !errCmp(err, tt.expected) {
				t.Errorf(cmp.Diff(err, tt.expected))
			}
		})
	}
}

func TestJobLevelRunner(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected map[string]error
	}{
		{
			name: "no runner defined",
			path: "./testdata/workflow-no-runner.yml",
			expected: map[string]error{
				"args": nil, "build": nil, "upload": nil,
			},
		},
		{
			name: "runners defined GH-hosted",
			path: "./testdata/workflow-gh-hosted-runners.yml",
			expected: map[string]error{
				"args": nil, "build": nil, "upload": nil,
			},
		},
		{
			name: "runner self-hosted first",
			path: "./testdata/workflow-first-self-hosted-runners.yml",
			expected: map[string]error{
				"args": errorSelfHostedRunner, "build": nil, "upload": nil,
			},
		},
		{
			name: "runner self-hosted second",
			path: "./testdata/workflow-second-self-hosted-runners.yml",
			expected: map[string]error{
				"args": nil, "build": errorSelfHostedRunner, "upload": nil,
			},
		},
		{
			name: "runner self-hosted third",
			path: "./testdata/workflow-third-self-hosted-runners.yml",
			expected: map[string]error{
				"args": nil, "build": nil, "upload": errorSelfHostedRunner,
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
				err = workflow.validateJobRunner(job)
				if !errCmp(err, val) {
					t.Errorf(cmp.Diff(err, val))
				}
			}
		})
	}
}

func TestJobLevelStep(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected map[string]error
	}{
		{
			name: "no steps defined",
			path: "./testdata/workflow-no-steps.yml",
			expected: map[string]error{
				"args": nil, "build": nil, "upload": nil,
			},
		},
		{
			name: "steps mix defined",
			path: "./testdata/workflow-job-steps-mix.yml",
			expected: map[string]error{
				"args":   errorDeclaredStep,
				"job2":   errorDeclaredStep,
				"job3":   nil,
				"job4":   errorDeclaredStep,
				"build":  nil,
				"upload": errorDeclaredStep,
				"job5":   nil,
				"job6":   errorDeclaredStep,
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
				err = workflow.validateJobSteps(job)
				if !errCmp(err, val) {
					t.Errorf(cmp.Diff(err, val))
				}
			}
		})
	}
}

type expectedStruct struct {
	ok  bool
	err error
}

func TestTrustedReusableWorkflow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected map[string]expectedStruct
	}{
		{
			name: "no re-usable workflow",
			path: "./testdata/workflow-no-reusable-workflow.yml",
			expected: map[string]expectedStruct{
				"args":   expectedStruct{ok: false, err: nil},
				"build":  expectedStruct{ok: false, err: nil},
				"upload": expectedStruct{ok: false, err: nil},
			},
		},
		{
			name: "re-usable workflow mix",
			path: "./testdata/workflow-reusable-workflow-mix.yml",
			expected: map[string]expectedStruct{
				"args":   expectedStruct{ok: false, err: nil},
				"job2":   expectedStruct{ok: false, err: nil},
				"job3":   expectedStruct{ok: false, err: nil},
				"job4":   expectedStruct{ok: false, err: nil},
				"build":  expectedStruct{ok: true, err: nil},
				"job5":   expectedStruct{ok: false, err: nil},
				"job6":   expectedStruct{ok: false, err: errorInvalidReUsableWorkflow},
				"job7":   expectedStruct{ok: false, err: errorInvalidReUsableWorkflow},
				"upload": expectedStruct{ok: false, err: nil},
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
				ok, err := workflow.isJobCallingTrustedReusableWorkflow(job)
				if !errCmp(err, val.err) {
					t.Errorf(cmp.Diff(err, val))
				}

				if ok != val.ok {
					t.Errorf(cmp.Diff(ok, val.ok))
				}
			}
		})
	}
}

func TestTopLevelPermissions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected error
	}{
		{
			name:     "no top level permissions defined",
			path:     "./testdata/workflow-no-top-permissions.yml",
			expected: errorPermissionsDefaultWrite,
		},
		{
			name:     "top level permissions empty",
			path:     "./testdata/workflow-top-permissions-empty.yml",
			expected: nil,
		},
		{
			name:     "top level permissions write-all",
			path:     "./testdata/workflow-top-permissions-writeall.yml",
			expected: errorPermissionsNotReadAll,
		},
		{
			name:     "top level permissions set contents:write",
			path:     "./testdata/workflow-top-permissions-contents-write.yml",
			expected: errorPermissionWrite,
		},
		{
			name:     "top level permissions set actions:write",
			path:     "./testdata/workflow-top-permissions-actions-write.yml",
			expected: errorPermissionWrite,
		},
		{
			name:     "top level permissions set id-token:write",
			path:     "./testdata/workflow-top-permissions-idtoken-write.yml",
			expected: errorPermissionWrite,
		},
		{
			name:     "top level permissions set other:write not dangerous",
			path:     "./testdata/workflow-top-permissions-other-write-no-dangerous.yml",
			expected: nil,
		},
		{
			name:     "top level permissions set other:write dangerous read",
			path:     "./testdata/workflow-top-permissions-other-write-dangerous-read.yml",
			expected: nil,
		},
		{
			name:     "top level permissions set other:write dangerous empty",
			path:     "./testdata/workflow-top-permissions-other-write-dangerous-empty.yml",
			expected: nil,
		},
		{
			name:     "top level permissions set other:write dangerous none",
			path:     "./testdata/workflow-top-permissions-other-write-dangerous-none.yml",
			expected: nil,
		},
		{
			name:     "top level permissions set other:write dangerous write",
			path:     "./testdata/workflow-top-permissions-other-write-dangerous-write.yml",
			expected: errorPermissionWrite,
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

			err = workflow.validateTopLevelPermissions()

			if !errCmp(err, tt.expected) {
				t.Errorf(cmp.Diff(err, tt.expected))
			}
		})
	}
}

func TestJobLevelPermissions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected map[string]error
	}{
		{
			name: "no job permissions",
			path: "./testdata/workflow-no-job-permissions.yml",
			expected: map[string]error{
				"args": nil, "build": nil, "upload": nil,
			},
		},
		{
			name: "job permission empty",
			path: "./testdata/workflow-job-permissions-empty.yml",
			expected: map[string]error{
				"args": nil, "build": nil, "upload": nil,
			},
		},
		{
			name: "job permission write-all",
			path: "./testdata/workflow-job-permissions-writeall.yml",
			expected: map[string]error{
				"args":   errorPermissionsNotReadAll,
				"build":  errorPermissionsNotReadAll,
				"upload": errorPermissionsNotReadAll,
			},
		},
		{
			name: "job permission mix write",
			path: "./testdata/workflow-job-permissions-mix-write.yml",
			expected: map[string]error{
				"args":   errorPermissionWrite,
				"build":  errorPermissionWrite,
				"upload": errorPermissionWrite,
			},
		},
		{
			name: "job permission others write no dangerous",
			path: "./testdata/workflow-job-permissions-others-write-no-dangerous.yml",
			expected: map[string]error{
				"args":   nil,
				"build":  nil,
				"upload": nil,
			},
		},
		{
			name: "job permission others write dangerous read",
			path: "./testdata/workflow-job-permissions-others-write-dangerous-read.yml",
			expected: map[string]error{
				"args":   nil,
				"build":  nil,
				"upload": nil,
			},
		},
		{
			name: "job permission others write dangerous none",
			path: "./testdata/workflow-job-permissions-others-write-dangerous-none.yml",
			expected: map[string]error{
				"args":   nil,
				"build":  nil,
				"upload": nil,
			},
		},
		{
			name: "job permission others write dangerous empty",
			path: "./testdata/workflow-job-permissions-others-write-dangerous-empty.yml",
			expected: map[string]error{
				"args":   nil,
				"build":  nil,
				"upload": nil,
			},
		},
		{
			name: "job permission others write dangerous write",
			path: "./testdata/workflow-job-permissions-others-write-dangerous-write.yml",
			expected: map[string]error{
				"args":   errorPermissionWrite,
				"build":  errorPermissionWrite,
				"upload": errorPermissionWrite,
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
				err = workflow.validateUntrustedJobLevelPermissions(job)
				if !errCmp(err, val) {
					t.Errorf(cmp.Diff(err, val))
				}
			}
		})
	}
}

func TestTrustedBuilderPermissions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		job      string
		expected error
	}{
		{
			name:     "correct job permissions",
			job:      "build",
			path:     "./testdata/workflow-trusted-job-correct-permissions.yml",
			expected: nil,
		},
		{
			name:     "job permissions contents write",
			job:      "build",
			path:     "./testdata/workflow-trusted-job-contents-write.yml",
			expected: errorInvalidPermission,
		},
		{
			name:     "job permissions contents empty",
			job:      "build",
			path:     "./testdata/workflow-trusted-job-contents-empty.yml",
			expected: errorInvalidPermission,
		},
		{
			name:     "job permissions contents none",
			job:      "build",
			path:     "./testdata/workflow-trusted-job-contents-none.yml",
			expected: errorInvalidPermission,
		},
		{
			name:     "job permissions id-token read",
			job:      "build",
			path:     "./testdata/workflow-trusted-job-idtoken-read.yml",
			expected: errorInvalidPermission,
		},
		{
			name:     "job permissions id-token empty",
			job:      "build",
			path:     "./testdata/workflow-trusted-job-idtoken-empty.yml",
			expected: errorInvalidPermission,
		},
		{
			name:     "job permissions id-token none",
			job:      "build",
			path:     "./testdata/workflow-trusted-job-idtoken-none.yml",
			expected: errorInvalidPermission,
		},
		{
			name:     "job permissions read-all",
			job:      "build",
			path:     "./testdata/workflow-trusted-job-read-all.yml",
			expected: errorPermissionAllSet,
		},
		{
			name:     "job permissions write-all",
			job:      "build",
			path:     "./testdata/workflow-trusted-job-write-all.yml",
			expected: errorPermissionAllSet,
		},
		{
			name:     "job permissions empty",
			job:      "build",
			path:     "./testdata/workflow-trusted-job-empty.yml",
			expected: errorPermissionAllSet,
		},
		{
			name:     "job permissions additional",
			job:      "build",
			path:     "./testdata/workflow-trusted-job-additional.yml",
			expected: errorPermissionScopeTooMany,
		},
		{
			name:     "job permissions no id-token scope",
			job:      "build",
			path:     "./testdata/workflow-trusted-job-no-idtoken-scopes.yml",
			expected: errorPermissionNotSet,
		},
		{
			name:     "job permissions no contents scope",
			job:      "build",
			path:     "./testdata/workflow-trusted-job-no-contents-scopes.yml",
			expected: errorPermissionNotSet,
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

			job, exists := workflow.workflow.Jobs[tt.job]
			if !exists {
				panic(fmt.Errorf("job not found in the workflow: %s", tt.job))
			}

			err = workflow.validateTrustedJobLevelPermissions(job)
			if !errCmp(err, tt.expected) {
				t.Errorf(cmp.Diff(err, tt.expected))
			}
		})
	}
}
