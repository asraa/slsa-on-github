package pkg

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rhysd/actionlint"
)

var errorInvalidGitHubWorkflow = errors.New("invalid GitHub workflow")

type Workflow struct {
	wf *actionlint.Workflow
}

func provenanceWorkflowFromString(content string) (*Workflow, error) {
	var pwf Workflow
	pwf.wf, errs := actionlint.Parse(content)
	if len(errs) > 0 && workflow == nil {
		return false, fileparser.FormatActionlintError(errs)
	}
	return &pwf, nil
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

	return fmt.Errorf("%w", builder.String())
}
