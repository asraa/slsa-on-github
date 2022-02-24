package pkg

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rhysd/actionlint"
)

var (
	errorInvalidGitHubWorkflow         = errors.New("invalid GitHub workflow")
	errorDeclaredEnv                   = errors.New("env variables are declared")
	errorDeclaredDefaults              = errors.New("defaults are declared")
	errorSelfHostedRunner              = errors.New("self-hosted runner not supported")
	errorDeclaredStep                  = errors.New("steps are declared")
	errorInvalidReUsableWorkflow       = errors.New("invalid re-usable workflow call")
	errorInvalidPermission             = errors.New("invalid permission")
	errorPermissionsDefaultWrite       = errors.New("no permission declared")
	errorPermissionsNotReadAll         = errors.New("permissions are not set to `read-all`")
	errorPermissionWrite               = errors.New("permission is set to write")
	errorInternalPermission            = errors.New("internal error parsing permissions")
	errorPermissionAllSet              = errors.New("permissions all set")
	errorPermissionScopeTooMany        = errors.New("too many permissions scopes defined")
	errorPermissionNotSet              = errors.New("permissions not set")
	errorMultipleJobsUseTrustedBuilder = errors.New("trusted builder used in multiple jobs")
	errorInternalUniqueJob             = errors.New("internal error retrieving trusted job")
	errorNoTrustedJobFound             = errors.New("no trusted job found")
)

// https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#choosing-github-hosted-runners.
var allowedRunners = map[string]bool{
	"ubuntu-latest": true, "ubuntu-20.04": true, "ubuntu-18.04": true,
}

var trustedReusableWorkflow = "asraa/slsa-on-github/.github/workflows/slsa-builder-go.yml"

var (
	permissionIdToken  = "id-token"
	permissionContents = "contents"
	permissionActions  = "actions"
)

// We allow `packages` to be set to `write` for caller to upload
// the package to GitHub registry. `contents` is not strictly needed,
// but we verify it anyway since it violates SLSA4.
var dangerousPermissions = map[string]bool{
	permissionContents: true, permissionIdToken: true,
	permissionActions: true,
}

type Workflow struct {
	workflow *actionlint.Workflow
}

func WorkflowFromBytes(content []byte) (*Workflow, error) {
	var self Workflow
	workflow, errs := actionlint.Parse(content)
	if len(errs) > 0 && workflow == nil {
		return nil, formatActionlintError(errs)
	}
	self.workflow = workflow
	return &self, nil
}

func (w *Workflow) Validate() error {
	// Verify the top-level constraints.
	if err := w.validateTopLevelDefinitions(); err != nil {
		return err
	}

	// Verify the job-level constraints.
	if err := w.validateJobLevelDefinitions(); err != nil {
		return err
	}

	return nil
}

func (w *Workflow) validateTopLevelDefinitions() error {
	// Defaults.
	// Note: not strictly necessary because a re-usable workflow
	// does not inherit the defaults.
	if err := w.validateTopLevelDefaults(); err != nil {
		return err
	}

	// Env variables.
	// Note: not strictly necessary because a re-usable workflow
	// does not inherit the env variables.
	if err := w.validateTopLevelEnv(); err != nil {
		return err
	}

	// Runner.
	// Note: not strictly necessary because self-hosted
	// runners cannot interfere with the re-usable workflow
	// that runs in its own VM. The call below includes
	// all jobs, including the trusted re-usable workflow.
	if err := w.validateRunners(); err != nil {
		return err
	}

	// Token permissions.
	// Note: this is needed.
	if err := w.validateTopLevelPermissions(); err != nil {
		return err
	}

	return nil
}

func (w *Workflow) validateJobLevelDefinitions() error {
	// Verify the trusted job definitions.
	if err := w.validateTrustedJobDefinitions(); err != nil {
		return err
	}

	// Verify other jobs' definitions.
	if err := w.validateUntrustedJobDefinitions(); err != nil {
		return err
	}
	return nil
}

func (w *Workflow) validateTrustedJobDefinitions() error {
	// Get the trusted builder, if it was niquely defined.
	trustedJob, err := w.getUniqueJobCallingTrustedReusableWorkflow()
	if err != nil {
		return err
	}

	if trustedJob == nil {
		return fmt.Errorf("%w", errorInternalUniqueJob)
	}

	// Runner.
	// Note: not necessary because re-usable workflows do not accept
	// runner labels defined in the calling workflow.
	if err := w.validateJobRunner(trustedJob); err != nil {
		return err
	}

	// Defaults.
	// Note: Not certain this is necessary, but verify it anyway.
	if err := w.validateJobLevelDefaults(trustedJob); err != nil {
		return err
	}

	// Env variables.
	// Note: not strictly necessary because re-usable workflows do not accept
	// env variables defined in the calling workflow.
	if err := w.validateTrustedReusableWorkflowEnv(trustedJob); err != nil {
		return err
	}

	// Steps.
	// Note: not strictly necessary because re-usable workflows do not accept
	// additional steps defined in the calling workflow.
	if err := w.validateTrustedReusableWorkflowSteps(trustedJob); err != nil {
		return err
	}

	// Permissions.
	// Note: this one is necessary.
	if err := w.validateTrustedReusableWorkflowPermissions(trustedJob); err != nil {
		return err
	}

	return nil
}

func (w *Workflow) validateUntrustedJobDefinitions() error {
	for _, job := range w.workflow.Jobs {
		if job == nil {
			continue
		}

		trusted, err := w.isJobCallingTrustedReusableWorkflow(job)
		if err != nil {
			return err
		}

		if trusted {
			continue
		}

		// Verify untrusted job.

		// Runner.
		// Note: not necessary because because other jobs cannot affect
		// the trusted re-usable workflow which runs in its own VM.
		if err := w.validateJobRunner(job); err != nil {
			return err
		}

		// Permissions.
		// Note: this one is necessary.
		if err := w.validateUntrustedJobLevelPermissions(job); err != nil {
			return err
		}
	}
	return nil
}

// =============== Defaults ================ //
func (w *Workflow) validateTopLevelDefaults() error {
	return validateDefaults(w.workflow.Defaults, "top level")
}

func (w *Workflow) validateJobLevelDefaults(job *actionlint.Job) error {
	return validateDefaults(job.Defaults, fmt.Sprintf("job %s", getJobIdentity(job)))
}

func validateDefaults(def *actionlint.Defaults, msg string) error {
	if def != nil {
		return fmt.Errorf("%s: %w", msg, errorDeclaredDefaults)
	}
	return nil
}

// =============== Env ================ //
func (w *Workflow) validateTopLevelEnv() error {
	return validateEnv(w.workflow.Env, "top level")
}

func (w *Workflow) validateTrustedReusableWorkflowEnv(job *actionlint.Job) error {
	return validateEnv(job.Env, fmt.Sprintf("job %s", getJobIdentity(job)))
}

func validateEnv(env *actionlint.Env, msg string) error {
	if env != nil && len(env.Vars) > 0 {
		return fmt.Errorf("%s: %w", msg, errorDeclaredEnv)
	}
	return nil
}

// =============== Runners ================ //
func (w *Workflow) validateRunners() error {
	for _, job := range w.workflow.Jobs {
		if job == nil {
			continue
		}

		if err := w.validateJobRunner(job); err != nil {
			return err
		}
	}

	return nil
}

func (w *Workflow) validateJobRunner(job *actionlint.Job) error {
	if err := validateJobRunner(job.RunsOn, allowedRunners); err != nil {
		return fmt.Errorf("%s: %w", fmt.Sprintf("job %s", getJobIdentity(job)), err)
	}
	return nil
}

func validateJobRunner(runner *actionlint.Runner, allowed map[string]bool) error {
	if runner == nil {
		return nil
	}

	for _, label := range runner.Labels {
		if label == nil {
			continue
		}

		if _, exists := allowed[label.Value]; !exists {
			return fmt.Errorf("%s: %w", label.Value, errorSelfHostedRunner)
		}

	}
	return nil
}

// =============== Steps ================ //
func (w *Workflow) validateTrustedReusableWorkflowSteps(job *actionlint.Job) error {
	for _, step := range job.Steps {
		if step == nil {
			continue
		}

		return fmt.Errorf("%s: %w", fmt.Sprintf("job %s", getJobIdentity(job)), errorDeclaredStep)
	}
	return nil
}

// =============== Re-usable workflow ================ //
func (w *Workflow) getUniqueJobCallingTrustedReusableWorkflow() (*actionlint.Job, error) {
	var rjob *actionlint.Job
	for _, job := range w.workflow.Jobs {
		if job == nil {
			continue
		}

		b, err := w.isJobCallingTrustedReusableWorkflow(job)
		if err != nil {
			return nil, err
		}

		if !b {
			continue
		}

		if rjob != nil {
			return nil, fmt.Errorf("%s: %w: %s", getJobIdentity(rjob), errorMultipleJobsUseTrustedBuilder, getJobIdentity(job))
		}

		rjob = job
	}

	if rjob == nil {
		return nil, fmt.Errorf("%w", errorNoTrustedJobFound)
	}

	return rjob, nil
}

func (w *Workflow) isJobCallingTrustedReusableWorkflow(job *actionlint.Job) (bool, error) {
	if job == nil || job.WorkflowCall == nil || job.WorkflowCall.Uses == nil {
		return false, nil
	}

	values := strings.Split(job.WorkflowCall.Uses.Value, "@")
	if len(values) != 2 {
		return false, fmt.Errorf("%s: %s is not pinned: %w", getJobIdentity(job),
			job.WorkflowCall.Uses.Value, errorInvalidReUsableWorkflow)
	}
	return strings.EqualFold(values[0], trustedReusableWorkflow), nil
}

// =============== Permissions ================ //
// https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/about-security-hardening-with-openid-connect.
func (w *Workflow) validateTopLevelPermissions() error {
	// No definition means all permissions are set write by default.
	if w.workflow.Permissions == nil {
		return fmt.Errorf("%w", errorPermissionsDefaultWrite)
	}

	return validateUntrustedPermissions(w.workflow.Permissions)
}

func (w *Workflow) validateUntrustedJobLevelPermissions(job *actionlint.Job) error {
	if job == nil {
		return nil
	}

	// Job permission not set inherit the top-level permissions,
	// which we validate via validateTopLevelPermissions().
	if job.Permissions == nil {
		return nil
	}

	if err := validateUntrustedPermissions(job.Permissions); err != nil {
		return fmt.Errorf("%s: %w", getJobIdentity(job), err)
	}

	return nil
}

func validateUntrustedPermissions(permissions *actionlint.Permissions) error {
	if permissions == nil {
		// Since this is different for top-level and job-level,
		// we should fail if this happens.
		return fmt.Errorf("%w", errorInternalPermission)
	}

	// `nil` value means `none` and is safe.
	// If it's not `nil`, we verify that permissions are set to `read-all` or `` (none all).
	if permissions.All != nil &&
		!strings.EqualFold(permissions.All.Value, "") &&
		!strings.EqualFold(permissions.All.Value, "read-all") {
		return fmt.Errorf("%w", errorPermissionsNotReadAll)
	}

	// Verify individual permissions set.
	for name, scope := range permissions.Scopes {
		if scope == nil || scope.Name == nil {
			return fmt.Errorf("%w: scope is nil", errorInvalidPermission)
		}

		if scope.Name.Value != name {
			return fmt.Errorf("%w: '%s' different from '%s'", errorInvalidPermission, scope.Name.Value, name)
		}

		// Note sure when this may happen, so returning an error to catch it.
		if scope.Value == nil {
			return fmt.Errorf("%w: %s: scope.Value is nil", errorInternalPermission, name)
		}

		// Value of permission is set: verify it is `read` or `none`.
		// We only verify certain permissions that are danegrous, including the
		// `id-token`, but we accept other permissions.
		// "" value means `none` and is safe.
		if isDangerousPermission(name) &&
			!strings.EqualFold(scope.Value.Value, "read") &&
			!strings.EqualFold(scope.Value.Value, "none") &&
			!strings.EqualFold(scope.Value.Value, "") {
			return fmt.Errorf("%s: %w", name, errorPermissionWrite)
		}

	}
	return nil
}

func (w *Workflow) validateTrustedReusableWorkflowPermissions(job *actionlint.Job) error {
	if job == nil {
		return fmt.Errorf("%w", errorInternalPermission)
	}

	// Permissions must be defined.
	if job.Permissions == nil {
		return fmt.Errorf("builder: %w", errorPermissionNotSet)
	}

	// No read-all.
	if job.Permissions.All != nil {
		return fmt.Errorf("builder: %w: %s", errorPermissionAllSet, job.Permissions.All.Value)
	}

	// Scopes defined.
	if len(job.Permissions.Scopes) != 2 {
		return fmt.Errorf("builder: %w", errorPermissionScopeTooMany)
	}

	// Validate the `id-token` permissions is set to `write`.
	if err := validateTrustedJobPermission(job.Permissions.Scopes, permissionIdToken, "write"); err != nil {
		return err
	}

	// Validate the `contents` permissions is set to `read`.
	// Note: this is only necessary for private repos.
	if err := validateTrustedJobPermission(job.Permissions.Scopes, permissionContents, "read"); err != nil {
		return err
	}

	return nil
}

func validateTrustedJobPermission(scopes map[string]*actionlint.PermissionScope,
	permissionName, permissionValue string) error {
	scope, exists := scopes[permissionName]
	if !exists {
		return fmt.Errorf("builder: %s: %w", permissionName, errorPermissionNotSet)
	}

	// Validate name.
	if scope == nil || scope.Name == nil {
		return fmt.Errorf("builder: %w: scope is nil", errorInvalidPermission)
	}

	if scope.Name.Value != permissionName {
		return fmt.Errorf("builder: %w: '%s' different from '%s'",
			errorInvalidPermission, scope.Name.Value, permissionName)
	}

	// Valdate value.
	if scope.Value == nil {
		return fmt.Errorf("builder: %s: %w: scope not set", permissionName, errorInvalidPermission)
	}

	if !strings.EqualFold(scope.Value.Value, permissionValue) {
		return fmt.Errorf("builder: %w: scope of %s is set to '%s'",
			errorInvalidPermission, permissionName, scope.Value.Value)
	}
	return nil
}

func isDangerousPermission(name string) bool {
	_, exists := dangerousPermissions[name]
	return exists
}

// =============== Utility ================ //
func getJobIdentity(job *actionlint.Job) string {
	var n string
	switch {
	case job.Name != nil:
		n = job.Name.Value
	case job.ID != nil:
		n = job.ID.Value
	default:
		n = "unknown-job"
	}
	return n
}

func formatActionlintError(errs []*actionlint.Error) error {
	if len(errs) == 0 {
		return nil
	}
	builder := strings.Builder{}
	builder.WriteString(errorInvalidGitHubWorkflow.Error() + ":")
	for _, err := range errs {
		builder.WriteString("\n" + err.Error())
	}

	return fmt.Errorf("%s", builder.String())
}
