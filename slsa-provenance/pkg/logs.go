// Copyright 2021 The slsa-on-github Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
A Workflow struct contains an ID, path, time informatio.
The WorkflowRun contains information on where that workflow was run (SHA, branch, event, jobs, logs, artifacts)
*/

package provenance

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/go-github/v40/github"
)

func GetCurrentWorkflowRunAndBuildJob(ctx context.Context, client *github.Client, org string, repo string, runId int64) (*github.Workflow, *github.WorkflowRun, *github.WorkflowJob, error) {
	workflowRun, resp, err := client.Actions.GetWorkflowRunByID(ctx, org, repo, runId)
	if err != nil {
		return nil, nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("returned status GetWorkflowRun %s\n", resp.Status)
		return nil, nil, nil, err
	}

	workflow, resp, err := client.Actions.GetWorkflowByID(ctx, org, repo, *workflowRun.WorkflowID)
	if err != nil {
		return nil, nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("returned status GetWorkflow %s\n", resp.Status)
		return nil, nil, nil, err
	}

	jobs, resp, err := client.Actions.ListWorkflowJobs(ctx, org, repo, runId, &github.ListWorkflowJobsOptions{})
	if err != nil {
		return nil, nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("returned status ListWorkflowJobs %s\n", resp.Status)
		return nil, nil, nil, err
	}

	for _, v := range jobs.Jobs {
		if strings.EqualFold(*v.Name, "build") {
			return workflow, workflowRun, v, nil
		}
	}
	return nil, nil, nil, errors.New("could not find job with name 'build'")
}

func GetBuildLogsURL(ctx context.Context, client *github.Client, org, repo string) (*url.URL, error) {
	runs, resp, err := client.Actions.ListRepositoryWorkflowRuns(ctx, org, repo, &github.ListWorkflowRunsOptions{})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("returned status workflow runs %s\n", resp.Status)
		return nil, err
	}

	workflow_run_id := runs.WorkflowRuns[0].ID

	if *runs.WorkflowRuns[0].Name != "SLSA Release" {
		fmt.Printf("name is actually %s", *runs.WorkflowRuns[0].Name)
	}

	job, resp, err := client.Actions.ListWorkflowJobs(ctx, org, repo, *workflow_run_id, &github.ListWorkflowJobsOptions{})
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("returned status  workflow job %s\n", resp.Status)
		return nil, err
	}

	job_id := job.Jobs[0].ID

	logURL, resp, err := client.Actions.GetWorkflowJobLogs(ctx, org, repo, *job_id, true)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusFound {
		fmt.Printf("returned status workflow job log %s\n", resp.Status)
		return nil, err
	}

	return logURL, nil
}

func GetBuildLogs(log_url *url.URL) ([]byte, error) {
	resp, err := http.Get(log_url.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	d, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return d, nil
}