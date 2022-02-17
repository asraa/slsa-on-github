package pkg

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rhysd/actionlint"
)

var (
	errorInvalidGitHubWorkflow = errors.New("invalid GitHub workflow")
	errorDeclaredEnv           = errors.New("env variables are declared")
	errorDeclaredDefaults      = errors.New("defaults are declared")
)

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

func (w *Workflow) validateJobLevelEnv(job *actionlint.Job) error {
	return validateEnv(job.Env, fmt.Sprintf("job %s", getJobIdentity(job)))
}

func validateEnv(env *actionlint.Env, msg string) error {
	if env != nil && len(env.Vars) > 0 {
		return fmt.Errorf("%s: %w", msg, errorDeclaredEnv)
	}
	return nil
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
