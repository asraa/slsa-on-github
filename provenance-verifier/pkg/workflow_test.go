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

func Test_validateTopLevelEnv(t *testing.T) {
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

func Test_validateTrustedReusableWorkflowEnv(t *testing.T) {
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
				err = workflow.validateTrustedReusableWorkflowEnv(job)
				if !errCmp(err, val) {
					t.Errorf(cmp.Diff(err, val))
				}
			}
		})
	}
}

func Test_validateJobLevelDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected map[string]error
	}{
		{
			name: "no job default",
			path: "./testdata/workflow-no-job-defaults.yml",
			expected: map[string]error{
				"args": nil, "build": nil, "upload": nil,
			},
		},
		{
			name: "job default defined",
			path: "./testdata/workflow-job-defaults.yml",
			expected: map[string]error{
				"args":   errorDeclaredDefaults,
				"build":  errorDeclaredDefaults,
				"upload": errorDeclaredDefaults,
			},
		},
		{
			name: "job default mix",
			path: "./testdata/workflow-job-defaults-mix.yml",
			expected: map[string]error{
				"args":   errorDeclaredDefaults,
				"job2":   errorDeclaredDefaults,
				"job3":   errorDeclaredDefaults,
				"job4":   nil,
				"build":  errorDeclaredDefaults,
				"upload": errorDeclaredDefaults,
				"job5":   nil,
				"job6":   nil,
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
				err = workflow.validateJobLevelDefaults(job)
				if !errCmp(err, val) {
					t.Errorf(cmp.Diff(err, val))
				}
			}
		})
	}
}

func Test_validateTopLevelDefaults(t *testing.T) {
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

func Test_validateRunners(t *testing.T) {
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

			err = workflow.validateRunners()
			if !errCmp(err, tt.expected) {
				t.Errorf(cmp.Diff(err, tt.expected))
			}
		})
	}
}

func Test_validateJobRunner(t *testing.T) {
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

func Test_validateTrustedReusableWorkflowSteps(t *testing.T) {
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
				err = workflow.validateTrustedReusableWorkflowSteps(job)
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

func Test_isJobCallingTrustedReusableWorkflow(t *testing.T) {
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

func Test_validateTopLevelPermissions(t *testing.T) {
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
			name:     "top level permissions set other empty dangerous empty",
			path:     "./testdata/workflow-top-permissions-other-empty-dangerous-empty.yml",
			expected: nil,
		},
		{
			name:     "top level permissions set other empty dangerous read",
			path:     "./testdata/workflow-top-permissions-other-empty-dangerous-read.yml",
			expected: nil,
		},
		{
			name:     "top level permissions set other empty contents write",
			path:     "./testdata/workflow-top-permissions-other-empty-contents-write.yml",
			expected: errorPermissionWrite,
		},
		{
			name:     "top level permissions set other empty id-token write",
			path:     "./testdata/workflow-top-permissions-other-empty-idtoken-write.yml",
			expected: errorPermissionWrite,
		},
		{
			name:     "top level permissions set other empty actions write",
			path:     "./testdata/workflow-top-permissions-other-empty-actions-write.yml",
			expected: errorPermissionWrite,
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

func Test_validateUntrustedJobLevelPermissions(t *testing.T) {
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
			name: "job permission others empty dangerous write",
			path: "./testdata/workflow-job-permissions-others-empty-dangerous-write.yml",
			expected: map[string]error{
				"args":   errorPermissionWrite,
				"build":  errorPermissionWrite,
				"upload": errorPermissionWrite,
			},
		},
		{
			name: "job permission others empty dangerous read",
			path: "./testdata/workflow-job-permissions-others-empty-dangerous-read.yml",
			expected: map[string]error{
				"args":   nil,
				"build":  nil,
				"upload": nil,
			},
		},
		{
			name: "job permission others empty dangerous empty",
			path: "./testdata/workflow-job-permissions-others-empty-dangerous-empty.yml",
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

func Test_validateTrustedJobDefinitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected error
	}{
		// Existence of the trusted job.
		{
			name:     "single trusted job",
			path:     "./testdata/workflow-trusted-job-definition-single.yml",
			expected: nil,
		},
		{
			name:     "two trusted job",
			path:     "./testdata/workflow-trusted-job-definition-two.yml",
			expected: errorMultipleJobsUseTrustedBuilder,
		},
		{
			name:     "no trusted job",
			path:     "./testdata/workflow-trusted-job-definition-none.yml",
			expected: errorNoTrustedJobFound,
		},
		// Runner.
		{
			name:     "no runner defined",
			path:     "./testdata/workflow-no-runner.yml",
			expected: nil,
		},
		{
			name:     "runners defined GH-hosted",
			path:     "./testdata/workflow-gh-hosted-runners-trusted.yml",
			expected: nil,
		},
		{
			name:     "runner self-hosted first",
			path:     "./testdata/workflow-first-self-hosted-runners-trusted.yml",
			expected: nil,
		},
		{
			name:     "runner self-hosted second",
			path:     "./testdata/workflow-second-self-hosted-runners-trusted.yml",
			expected: nil,
		},
		{
			name:     "runner self-hosted third",
			path:     "./testdata/workflow-third-self-hosted-runners-trusted.yml",
			expected: nil,
		},
		// Env.
		// Note: env variables cannot be declared for a job that calls a re-usable workflow.
		{
			name:     "no job level env",
			path:     "./testdata/workflow-no-job-env-trusted.yml",
			expected: nil,
		},
		{
			name:     "job level env defined but empty",
			path:     "./testdata/workflow-job-env-empty-trusted.yml",
			expected: nil,
		},
		{
			name:     "job level env defined one set single empty",
			path:     "./testdata/workflow-job-env-one-set-single-trusted.yml",
			expected: nil,
		},
		{
			name:     "job level env defined and set",
			path:     "./testdata/workflow-job-env-set-trusted.yml",
			expected: nil,
		},
		{
			name:     "job level env mix",
			path:     "./testdata/workflow-job-env-mix-trusted.yml",
			expected: nil,
		},
		// Steps.
		// Note: a job calling a re-usable workflow cannot have steps defined.
		{
			name:     "no steps defined",
			path:     "./testdata/workflow-no-steps-trusted.yml",
			expected: nil,
		},
		{
			name:     "steps mix defined",
			path:     "./testdata/workflow-job-steps-mix-trusted.yml",
			expected: nil,
		},
		// Defaults.
		// Note: default cannot be declared in a job that calls a re-usable workflow.
		{
			name:     "no job default",
			path:     "./testdata/workflow-no-job-defaults-trusted.yml",
			expected: nil,
		},
		{
			name:     "job default mix",
			path:     "./testdata/workflow-job-defaults-mix-trusted.yml",
			expected: nil,
		},
		// Permissions.
		{
			name:     "three scopes",
			path:     "./testdata/workflow-trusted-three-scopes.yml",
			expected: errorPermissionScopeInvalidNumber,
		},
		{
			name:     "no scopes",
			path:     "./testdata/workflow-trusted-no-scopes.yml",
			expected: errorPermissionAllSet,
		},
		{
			name:     "one scope",
			path:     "./testdata/workflow-trusted-one-scope.yml",
			expected: errorPermissionScopeInvalidNumber,
		},
		{
			name:     "correct job permissions",
			path:     "./testdata/workflow-trusted-job-correct-permissions-trusted.yml",
			expected: nil,
		},
		{
			name:     "job permissions contents write",
			path:     "./testdata/workflow-trusted-job-contents-write-trusted.yml",
			expected: errorInvalidPermission,
		},
		{
			name:     "job permissions contents empty",
			path:     "./testdata/workflow-trusted-job-contents-empty-trusted.yml",
			expected: errorInvalidPermission,
		},
		{
			name:     "job permissions contents none",
			path:     "./testdata/workflow-trusted-job-contents-none-trusted.yml",
			expected: errorInvalidPermission,
		},
		{
			name:     "job permissions id-token read",
			path:     "./testdata/workflow-trusted-job-idtoken-read-trusted.yml",
			expected: errorInvalidPermission,
		},
		{
			name:     "job permissions id-token empty",
			path:     "./testdata/workflow-trusted-job-idtoken-empty-trusted.yml",
			expected: errorInvalidPermission,
		},
		{
			name:     "job permissions id-token none",
			path:     "./testdata/workflow-trusted-job-idtoken-none-trusted.yml",
			expected: errorInvalidPermission,
		},
		{
			name:     "job permissions read-all",
			path:     "./testdata/workflow-trusted-job-read-all-trusted.yml",
			expected: errorPermissionAllSet,
		},
		{
			name:     "job permissions write-all",
			path:     "./testdata/workflow-trusted-job-write-all-trusted.yml",
			expected: errorPermissionAllSet,
		},
		{
			name:     "job permissions empty",
			path:     "./testdata/workflow-trusted-job-empty-trusted.yml",
			expected: errorPermissionAllSet,
		},
		{
			name:     "job permissions additional",
			path:     "./testdata/workflow-trusted-job-additional-trusted.yml",
			expected: errorPermissionScopeInvalidNumber,
		},
		{
			name:     "job permissions no id-token scope",
			path:     "./testdata/workflow-trusted-job-no-idtoken-scopes-trusted.yml",
			expected: errorPermissionNotSet,
		},
		{
			name:     "job permissions no contents scope",
			path:     "./testdata/workflow-trusted-job-no-contents-scopes-trusted.yml",
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

			err = workflow.validateTrustedJobDefinitions()
			if !errCmp(err, tt.expected) {
				t.Errorf(cmp.Diff(err, tt.expected))
			}
		})
	}
}

func Test_validateTrustedReusableWorkflowPermissions(t *testing.T) {
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
			expected: errorPermissionScopeInvalidNumber,
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

			err = workflow.validateTrustedReusableWorkflowPermissions(job)
			if !errCmp(err, tt.expected) {
				t.Errorf(cmp.Diff(err, tt.expected))
			}
		})
	}
}

func Test_getUniqueJobCallingTrustedReusableWorkflow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		job      string
		expected error
	}{
		{
			name:     "single trusted job",
			job:      "build",
			path:     "./testdata/workflow-trusted-job-definition-single.yml",
			expected: nil,
		},
		{
			name:     "two trusted job",
			job:      "build",
			path:     "./testdata/workflow-trusted-job-definition-two.yml",
			expected: errorMultipleJobsUseTrustedBuilder,
		},
		{
			name:     "no trusted job",
			path:     "./testdata/workflow-trusted-job-definition-none.yml",
			expected: errorNoTrustedJobFound,
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

			job, err := workflow.getUniqueJobCallingTrustedReusableWorkflow()
			if !errCmp(err, tt.expected) {
				t.Errorf(cmp.Diff(err, tt.expected))
			}

			if err != nil || tt.job == "" {
				return
			}

			expectedJob, exists := workflow.workflow.Jobs[tt.job]
			if !exists {
				panic(fmt.Errorf("job not found in the workflow: %s", tt.job))
			}

			if job == nil && expectedJob != nil {
				t.Errorf("job is nil but expectedJob is not")
			}

			if job != nil && expectedJob == nil {
				t.Errorf("job is not nil but expectedJob not")
			}

			if job != nil && (job.ID == nil || job.ID.Value != tt.job) {
				t.Errorf("%v != %s", job.ID, tt.job)
			}
		})
	}
}

func Test_validateTopLevelDefinitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected error
	}{
		// Top-level defaults.
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
		// Top-level Env.
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
		// Runners.
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
		// Top-level permissions.
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
			name:     "top level permissions set other empty dangerous empty",
			path:     "./testdata/workflow-top-permissions-other-empty-dangerous-empty.yml",
			expected: nil,
		},
		{
			name:     "top level permissions set other empty dangerous read",
			path:     "./testdata/workflow-top-permissions-other-empty-dangerous-read.yml",
			expected: nil,
		},
		{
			name:     "top level permissions set other empty contents write",
			path:     "./testdata/workflow-top-permissions-other-empty-contents-write.yml",
			expected: errorPermissionWrite,
		},
		{
			name:     "top level permissions set other empty id-token write",
			path:     "./testdata/workflow-top-permissions-other-empty-idtoken-write.yml",
			expected: errorPermissionWrite,
		},
		{
			name:     "top level permissions set other empty actions write",
			path:     "./testdata/workflow-top-permissions-other-empty-actions-write.yml",
			expected: errorPermissionWrite,
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

			err = workflow.validateTopLevelDefinitions()
			if !errCmp(err, tt.expected) {
				t.Errorf(cmp.Diff(err, tt.expected))
			}
		})
	}
}

func Test_validateUntrustedJobDefinitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected error
	}{
		// Runners.
		{
			name:     "no runner defined",
			path:     "./testdata/workflow-no-runner.yml",
			expected: nil,
		},
		{
			name:     "runners defined GH-hosted",
			path:     "./testdata/workflow-gh-hosted-runners-reusable.yml",
			expected: nil,
		},
		{
			name:     "runner self-hosted first",
			path:     "./testdata/workflow-first-self-hosted-runners-reusable.yml",
			expected: errorSelfHostedRunner,
		},
		{
			name:     "runner self-hosted second",
			path:     "./testdata/workflow-second-self-hosted-runners.yml",
			expected: errorSelfHostedRunner,
		},
		{
			name:     "runner self-hosted third",
			path:     "./testdata/workflow-third-self-hosted-runners-reusable.yml",
			expected: errorSelfHostedRunner,
		},
		// Permissions.
		{
			name:     "no job permissions",
			path:     "./testdata/workflow-no-job-permissions.yml",
			expected: nil,
		},
		{
			name:     "job permission empty",
			path:     "./testdata/workflow-job-permissions-empty.yml",
			expected: nil,
		},
		{
			name:     "job permission write-all",
			path:     "./testdata/workflow-job-permissions-writeall.yml",
			expected: errorPermissionsNotReadAll,
		},
		{
			name:     "job permission mix write",
			path:     "./testdata/workflow-job-permissions-mix-write.yml",
			expected: errorPermissionWrite,
		},
		{
			name:     "job permission others write no dangerous",
			path:     "./testdata/workflow-job-permissions-others-write-no-dangerous.yml",
			expected: nil,
		},
		{
			name:     "job permission others empty dangerous write",
			path:     "./testdata/workflow-job-permissions-others-empty-dangerous-write.yml",
			expected: errorPermissionWrite,
		},
		{
			name:     "job permission others empty dangerous read",
			path:     "./testdata/workflow-job-permissions-others-empty-dangerous-read.yml",
			expected: nil,
		},
		{
			name:     "job permission others empty dangerous empty",
			path:     "./testdata/workflow-job-permissions-others-empty-dangerous-empty.yml",
			expected: nil,
		},
		{
			name:     "job permission others write dangerous read",
			path:     "./testdata/workflow-job-permissions-others-write-dangerous-read.yml",
			expected: nil,
		},
		{
			name:     "job permission others write dangerous none",
			path:     "./testdata/workflow-job-permissions-others-write-dangerous-none.yml",
			expected: nil,
		},
		{
			name:     "job permission others write dangerous empty",
			path:     "./testdata/workflow-job-permissions-others-write-dangerous-empty.yml",
			expected: nil,
		},
		{
			name:     "job permission others write dangerous write",
			path:     "./testdata/workflow-job-permissions-others-write-dangerous-write.yml",
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
			err = workflow.validateUntrustedJobDefinitions()
			if !errCmp(err, tt.expected) {
				t.Errorf(cmp.Diff(err, tt.expected))
			}
		})
	}
}
