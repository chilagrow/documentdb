// Copyright 2021 FerretDB Inc.
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

package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/sethvargo/go-githubactions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getEnvFunc implements [os.Getenv] for testing.
func getEnvFunc(t *testing.T, env map[string]string) func(string) string {
	t.Helper()

	return func(key string) string {
		val, ok := env[key]
		require.True(t, ok, "missing key %q", key)

		return val
	}
}

func TestDefine(t *testing.T) {
	for name, tc := range map[string]struct {
		env       map[string]string
		pgVersion string
		expected  *result
	}{
		"pull_request": {
			env: map[string]string{
				"GITHUB_BASE_REF":   "ferretdb",
				"GITHUB_EVENT_NAME": "pull_request",
				"GITHUB_HEAD_REF":   "docker-tag",
				"GITHUB_REF_NAME":   "1/merge",
				"GITHUB_REF_TYPE":   "branch",
				"GITHUB_REPOSITORY": "FerretDB/documentdb",
			},
			expected: &result{
				developmentImages: []string{
					"ghcr.io/ferretdb/documentdb-dev:pr-docker-tag",
				},
			},
		},

		"pull_request_target": {
			env: map[string]string{
				"GITHUB_BASE_REF":   "main",
				"GITHUB_EVENT_NAME": "pull_request_target",
				"GITHUB_HEAD_REF":   "docker-tag",
				"GITHUB_REF_NAME":   "ferretdb",
				"GITHUB_REF_TYPE":   "branch",
				"GITHUB_REPOSITORY": "FerretDB/FerretDB",
			},
			expected: &result{
				developmentImages: []string{
					"ghcr.io/ferretdb/ferretdb-dev:pr-docker-tag",
				},
			},
		},

		"push/main": {
			env: map[string]string{
				"GITHUB_BASE_REF":   "",
				"GITHUB_EVENT_NAME": "push",
				"GITHUB_HEAD_REF":   "",
				"GITHUB_REF_NAME":   "main",
				"GITHUB_REF_TYPE":   "branch",
				"GITHUB_REPOSITORY": "FerretDB/FerretDB",
			},
		},

		"push/tag/release": {
			env: map[string]string{
				"GITHUB_BASE_REF":   "",
				"GITHUB_EVENT_NAME": "push",
				"GITHUB_HEAD_REF":   "",
				"GITHUB_REF_NAME":   "v0.102.0-ferretdb",
				"GITHUB_REF_TYPE":   "tag",
				"GITHUB_REPOSITORY": "FerretDB/documentdb",
			},
			pgVersion: "16",
			expected: &result{
				developmentImages: []string{
					"ferretdb/documentdb-dev:16-0.102.0-ferretdb",
					"ghcr.io/ferretdb/documentdb-dev:16-0.102.0-ferretdb",
					"quay.io/ferretdb/documentdb-dev:16-0.102.0-ferretdb",
				},
				productionImages: []string{
					"ferretdb/documentdb:16-0.102.0-ferretdb",
					"ghcr.io/ferretdb/documentdb:16-0.102.0-ferretdb",
					"quay.io/ferretdb/documentdb:16-0.102.0-ferretdb",
				},
			},
		},

		"push/tag/release-rc-major-minor": {
			env: map[string]string{
				"GITHUB_BASE_REF":   "",
				"GITHUB_EVENT_NAME": "push",
				"GITHUB_HEAD_REF":   "",
				"GITHUB_REF_NAME":   "v0.102.0-ferretdb-2.0.0-rc2",
				"GITHUB_REF_TYPE":   "tag",
				"GITHUB_REPOSITORY": "FerretDB/FerretDB",
			},
			pgVersion: "16.7", // set major and minor version of Postgresql
			expected: &result{
				developmentImages: []string{
					"ferretdb/documentdb-dev:16-0.102.0-ferretdb-2.0.0-rc2",
					"ferretdb/documentdb-dev:16.7-0.102.0-ferretdb-2.0.0-rc2",
					"ferretdb/documentdb-dev:latest",
					"ghcr.io/ferretdb/documentdb-dev:16-0.102.0-ferretdb-2.0.0-rc2",
					"ghcr.io/ferretdb/documentdb-dev:16.7-0.102.0-ferretdb-2.0.0-rc2",
					"ghcr.io/ferretdb/documentdb-dev:latest",
					"quay.io/ferretdb/documentdb-dev:16-0.102.0-ferretdb-2.0.0-rc2",
					"quay.io/ferretdb/documentdb-dev:16.7-0.102.0-ferretdb-2.0.0-rc2",
					"quay.io/ferretdb/documentdb-dev:latest",
				},
				productionImages: []string{
					"ferretdb/documentdb:16-0.102.0-ferretdb-2.0.0-rc2",
					"ferretdb/documentdb:16.7-0.102.0-ferretdb-2.0.0-rc2",
					"ferretdb/documentdb:latest",
					"ghcr.io/ferretdb/documentdb:16-0.102.0-ferretdb-2.0.0-rc2",
					"ghcr.io/ferretdb/documentdb:16.7-0.102.0-ferretdb-2.0.0-rc2",
					"ghcr.io/ferretdb/documentdb:latest",
					"quay.io/ferretdb/documentdb:16-0.102.0-ferretdb-2.0.0-rc2",
					"quay.io/ferretdb/documentdb:16.7-0.102.0-ferretdb-2.0.0-rc2",
					"quay.io/ferretdb/documentdb:latest",
				},
			},
		},

		"push/tag/release-major": {
			env: map[string]string{
				"GITHUB_BASE_REF":   "",
				"GITHUB_EVENT_NAME": "push",
				"GITHUB_HEAD_REF":   "",
				"GITHUB_REF_NAME":   "v0.102.0-ferretdb-2.0.0-rc2",
				"GITHUB_REF_TYPE":   "tag",
				"GITHUB_REPOSITORY": "FerretDB/FerretDB",
			},
			pgVersion: "16", // set major version of Postgresql
			expected: &result{
				developmentImages: []string{
					"ferretdb/documentdb-dev:16-0.102.0-ferretdb-2.0.0-rc2",
					"ferretdb/documentdb-dev:latest",
					"ghcr.io/ferretdb/documentdb-dev:16-0.102.0-ferretdb-2.0.0-rc2",
					"ghcr.io/ferretdb/documentdb-dev:latest",
					"quay.io/ferretdb/documentdb-dev:16-0.102.0-ferretdb-2.0.0-rc2",
					"quay.io/ferretdb/documentdb-dev:latest",
				},
				productionImages: []string{
					"ferretdb/documentdb:16-0.102.0-ferretdb-2.0.0-rc2",
					"ferretdb/documentdb:latest",
					"ghcr.io/ferretdb/documentdb:16-0.102.0-ferretdb-2.0.0-rc2",
					"ghcr.io/ferretdb/documentdb:latest",
					"quay.io/ferretdb/documentdb:16-0.102.0-ferretdb-2.0.0-rc2",
					"quay.io/ferretdb/documentdb:latest",
				},
			},
		},

		"push/tag/wrong": {
			env: map[string]string{
				"GITHUB_BASE_REF":   "",
				"GITHUB_EVENT_NAME": "push",
				"GITHUB_HEAD_REF":   "",
				"GITHUB_REF_NAME":   "0.102.0-ferretdb-2.0.0-rc2", // no leading v
				"GITHUB_REF_TYPE":   "tag",
				"GITHUB_REPOSITORY": "FerretDB/FerretDB",
			},
		},

		"schedule": {
			env: map[string]string{
				"GITHUB_BASE_REF":   "",
				"GITHUB_EVENT_NAME": "schedule",
				"GITHUB_HEAD_REF":   "",
				"GITHUB_REF_NAME":   "ferretdb",
				"GITHUB_REF_TYPE":   "branch",
				"GITHUB_REPOSITORY": "FerretDB/FerretDB",
			},
			expected: &result{
				developmentImages: []string{
					"ferretdb/documentdb-dev:branch-ferretdb",
					"ghcr.io/ferretdb/documentdb-dev:branch-ferretdb",
					"quay.io/ferretdb/documentdb-dev:branch-ferretdb",
				},
			},
		},

		"workflow_run": {
			env: map[string]string{
				"GITHUB_BASE_REF":   "",
				"GITHUB_EVENT_NAME": "workflow_run",
				"GITHUB_HEAD_REF":   "",
				"GITHUB_REF_NAME":   "ferretdb",
				"GITHUB_REF_TYPE":   "branch",
				"GITHUB_REPOSITORY": "FerretDB/FerretDB",
			},
			expected: &result{
				developmentImages: []string{
					"ferretdb/documentdb-dev:branch-ferretdb",
					"ghcr.io/ferretdb/documentdb-dev:branch-ferretdb",
					"quay.io/ferretdb/documentdb-dev:branch-ferretdb",
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			actual, err := define(getEnvFunc(t, tc.env))
			if tc.expected == nil {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestImageURL(t *testing.T) {
	// expected URLs should work
	assert.Equal(
		t,
		"https://ghcr.io/ferretdb/documentdb-dev:pr-docker-tag",
		imageURL("ghcr.io/ferretdb/documentdb-dev:pr-docker-tag"),
	)
	assert.Equal(
		t,
		"https://quay.io/ferretdb/documentdb-dev:pr-docker-tag",
		imageURL("quay.io/ferretdb/documentdb-dev:pr-docker-tag"),
	)
	assert.Equal(
		t,
		"https://hub.docker.com/r/ferretdb/documentdb-dev/tags",
		imageURL("ferretdb/documentdb-dev:pr-docker-tag"),
	)
}

func TestResults(t *testing.T) {
	dir := t.TempDir()

	summaryF, err := os.CreateTemp(dir, "summary")
	require.NoError(t, err)
	defer summaryF.Close()

	outputF, err := os.CreateTemp(dir, "output")
	require.NoError(t, err)
	defer outputF.Close()

	var stdout bytes.Buffer
	getenv := getEnvFunc(t, map[string]string{
		"GITHUB_STEP_SUMMARY": summaryF.Name(),
		"GITHUB_OUTPUT":       outputF.Name(),
	})
	action := githubactions.New(githubactions.WithGetenv(getenv), githubactions.WithWriter(&stdout))

	result := &result{
		developmentImages: []string{
			"ghcr.io/ferretdb/documentdb-dev:16-0.102.0-ferretdb",
		},
		productionImages: []string{
			"quay.io/ferretdb/documentdb:latest",
		},
	}

	setResults(action, result)

	expectedStdout := strings.ReplaceAll(`
 |Type        |Image                                                                                                                |
 |----        |-----                                                                                                                |
 |Development |['ghcr.io/ferretdb/documentdb-dev:16-0.102.0-ferretdb'](https://ghcr.io/ferretdb/documentdb-dev:16-0.102.0-ferretdb) |
 |Production  |['quay.io/ferretdb/documentdb:latest'](https://quay.io/ferretdb/documentdb:latest)                                   |

`[1:], "'", "`",
	)
	assert.Equal(t, expectedStdout, stdout.String(), "stdout does not match")

	expectedSummary := strings.ReplaceAll(`
 |Type        |Image                                                                                                                |
 |----        |-----                                                                                                                |
 |Development |['ghcr.io/ferretdb/documentdb-dev:16-0.102.0-ferretdb'](https://ghcr.io/ferretdb/documentdb-dev:16-0.102.0-ferretdb) |
 |Production  |['quay.io/ferretdb/documentdb:latest'](https://quay.io/ferretdb/documentdb:latest)                                   |

`[1:], "'", "`",
	)
	b, err := io.ReadAll(summaryF)
	require.NoError(t, err)
	assert.Equal(t, expectedSummary, string(b), "summary does not match")

	expectedOutput := `
development_images<<_GitHubActionsFileCommandDelimeter_
ghcr.io/ferretdb/documentdb-dev:16-0.102.0-ferretdb
_GitHubActionsFileCommandDelimeter_
production_images<<_GitHubActionsFileCommandDelimeter_
quay.io/ferretdb/documentdb:latest
_GitHubActionsFileCommandDelimeter_
`[1:]
	b, err = io.ReadAll(outputF)
	require.NoError(t, err)
	assert.Equal(t, expectedOutput, string(b), "output parameters does not match")
}
