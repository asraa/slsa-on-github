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

package provenance

import (
	"fmt"

	"github.com/google/go-github/v40/github"
	intoto "github.com/in-toto/in-toto-golang/in_toto"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
)

// GenerateAttestation translates workflow run logs into a SLSA provenance
// attestation.
// Spec: https://slsa.dev/provenance/v0.1
func GenerateAttestation(workflow *github.Workflow, workflowRun *github.WorkflowRun, job *github.WorkflowJob, digest string) (intoto.ProvenanceStatement, error) {
	// Only the Job has information on the runners used for the build job in the run.
	att := intoto.ProvenanceStatement{
		StatementHeader: intoto.StatementHeader{
			Type:          intoto.StatementInTotoV01,
			PredicateType: slsa.PredicateSLSAProvenance,
			Subject: []intoto.Subject{
				{
					Name: "_",
					Digest: slsa.DigestSet{
						"sha256": digest,
					},
				},
			},
		},
		Predicate: slsa.ProvenancePredicate{
			BuildType: "https://github.com/Attestations/GitHubActionsWorkflow@v1",
			Builder: slsa.ProvenanceBuilder{
				ID: "https://github.com/Attestations/GitHubHostedActions@v1",
			},
			Invocation: slsa.ProvenanceInvocation{
				ConfigSource: slsa.ConfigSource{
					EntryPoint: fmt.Sprintf("%s%s", workflow.GetPath(), job.GetName()),
					URI:        fmt.Sprintf("git+%s.git", *workflowRun.HeadRepository.HTMLURL),
					Digest: slsa.DigestSet{
						"SHA1": workflowRun.GetHeadSHA(),
					},
				},
				Environment: map[string]interface{}{
					"arch": "amd64", // TODO: Does GitHub run actually expose this?
					"env": map[string]string{
						"GITHUB_RUN_NUMBER": fmt.Sprintf("%d", workflowRun.GetRunNumber()),
						"GITHUB_RUN_ID":     fmt.Sprintf("%d", job.GetRunID()),
						"GITHUB_EVENT_NAME": workflowRun.GetEvent(),
					},
				},
			},
			Materials: []slsa.ProvenanceMaterial{{
				URI: fmt.Sprintf("git+%s.git", *workflowRun.HeadRepository.HTMLURL),
				Digest: slsa.DigestSet{
					"SHA1": workflowRun.GetHeadSHA(),
				}},
			},
		},
	}
	return att, nil
}
