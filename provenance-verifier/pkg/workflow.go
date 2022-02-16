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

func (w *Workflow) validateTopLevelEnv() error {
	return validateEnv(w.workflow.Env, "top level")
}

func (w *Workflow) validatJobLevelEnv(job *actionlint.Job) error {
	var n string
	switch {
	case job.Name != nil:
		n = job.Name.Value
	case job.ID != nil:
		n = job.ID.Value
	default:
		n = "unknown-job"
	}

	return validateEnv(job.Env, fmt.Sprintf("job %s", n))
}

func validateEnv(env *actionlint.Env, msg string) error {
	if env != nil && len(env.Vars) > 0 {
		return fmt.Errorf("%s: %w", msg, errorDeclaredEnv)
	}
	return nil
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
