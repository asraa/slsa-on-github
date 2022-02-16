package pkg

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rhysd/actionlint"
)

var (
	errorInvalidGitHubWorkflow = errors.New("invalid GitHub workflow")
	errorTopLevelEnvVariables  = errors.New("top level env variables are declared")
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

func (w *Workflow) validateTopLevelEnvironmentVariables() error {
	if w.workflow.Env != nil && len(w.workflow.Env.Vars) > 0 {
		return fmt.Errorf("%w", errorTopLevelEnvVariables)
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
