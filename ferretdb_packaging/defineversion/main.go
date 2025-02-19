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

// Package main defines debian version number of DocumentDB for CI builds.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/sethvargo/go-githubactions"
)

func main() {
	controlFileF := flag.String("control-file", "../pg_documentdb/documentdb.control", "pg_documentdb/documentdb.control file path")

	flag.Parse()

	action := githubactions.New()

	if *controlFileF == "" {
		action.Fatalf("%s", "-control-file flag is empty.")
	}

	controlDefaultVersion, err := readControlDefaultVersion(*controlFileF)
	if err != nil {
		action.Fatalf("%s", err)
	}

	debugEnv(action)

	upstreamVersion, err := define(controlDefaultVersion, action.Getenv)
	if err != nil {
		action.Fatalf("%s", err)
	}

	setResults(action, upstreamVersion)
}

// controlDefaultVer matches major, minor and patch from default_version field in control file,
// see pg_documentdb_core/documentdb_core.control.
var controlDefaultVer = regexp.MustCompile(`^default_version\s=\s'(?P<major>[0-9]+)\.(?P<minor>[0-9]+)-(?P<patch>[0-9]+)'$`)

// tagVer is the syntax used by tags such as `v0.100.0`.
// It may contain additional string such as `v0.100.0-ferretdb` and `v0.100.0-ferretdb-2.0.1`.
//
//nolint:lll // for readibility
var tagVer = regexp.MustCompile(`^v(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)-?(?P<rest>[0-9a-zA-Z-.]+)?$`)

// disallowedVer matches disallowed characters of upstream version,
// see https://www.debian.org/doc/debian-policy/ch-controlfields.html#version.
var disallowedVer = regexp.MustCompile(`[^A-Za-z0-9~.+]`)

// debugEnv logs all environment variables that start with `GITHUB_` or `INPUT_`
// in debug level.
func debugEnv(action *githubactions.Action) {
	res := make([]string, 0, 30)

	for _, l := range os.Environ() {
		if strings.HasPrefix(l, "GITHUB_") || strings.HasPrefix(l, "INPUT_") {
			res = append(res, l)
		}
	}

	slices.Sort(res)

	action.Debugf("Dumping environment variables:")

	for _, l := range res {
		action.Debugf("\t%s", l)
	}
}

// readControlDefaultVersion reads the default_version field from the control file,
// and returns the control default version in SemVer format,
// see pg_documentdb_core/documentdb_core.control.
func readControlDefaultVersion(f string) (string, error) {
	file, err := os.Open(f)
	if err != nil {
		return "", fmt.Errorf("failed to read control file: %w", err)
	}

	defer file.Close()

	r := bufio.NewReader(file)

	for {
		var line []byte
		line, _, err = r.ReadLine()
		if err != nil {
			return "", fmt.Errorf("control file did not find default_version: %w", err)
		}

		match := controlDefaultVer.FindSubmatch(line)
		if match == nil || len(match) != controlDefaultVer.NumSubexp()+1 {
			continue
		}

		major := match[tagVer.SubexpIndex("major")]
		minor := match[tagVer.SubexpIndex("minor")]
		patch := match[tagVer.SubexpIndex("patch")]

		return fmt.Sprintf("%s.%s.%s", major, minor, patch), nil
	}
}

// define returns the upstream version using the environment variables of GitHub Actions.
// The upstream version does not allow `-` character and replaced with `~`.
//
// For release tags, it uses the tag name as the upstream version.
// For pull requests, branches and other GitHub Actions it uses
// control default version for upstream version prefix as it must start with digit.
//
// See upstream version in https://www.debian.org/doc/debian-policy/ch-controlfields.html#version.
func define(controlDefaultVersion string, getenv githubactions.GetenvFunc) (string, error) {
	var upstreamVersion string
	var err error

	switch event := getenv("GITHUB_EVENT_NAME"); event {
	case "pull_request", "pull_request_target":
		branch := strings.ToLower(getenv("GITHUB_HEAD_REF"))
		upstreamVersion = defineForPR(controlDefaultVersion, branch)

	case "push", "schedule", "workflow_run":
		refName := strings.ToLower(getenv("GITHUB_REF_NAME"))

		switch refType := strings.ToLower(getenv("GITHUB_REF_TYPE")); refType {
		case "branch":
			upstreamVersion, err = defineForBranch(controlDefaultVersion, refName)

		case "tag":
			upstreamVersion, err = defineForTag(refName)

		default:
			err = fmt.Errorf("unhandled ref type %q for event %q", refType, event)
		}

	default:
		err = fmt.Errorf("unhandled event type %q", event)
	}

	if err != nil {
		return "", err
	}

	if upstreamVersion == "" {
		return "", fmt.Errorf("both upstreamVersion and err are nil")
	}

	return upstreamVersion, nil
}

// defineForPR defines debian upstream version for pull requests.
// It replaces special characters in branch name with character `~`.
func defineForPR(controlDefaultVersion, branch string) string {
	// for branches like "dependabot/submodules/XXX"
	parts := strings.Split(branch, "/")
	branch = parts[len(parts)-1]
	branch = disallowedVer.ReplaceAllString(branch, "~")

	return fmt.Sprintf("%s~pr~%s", controlDefaultVersion, branch)
}

// defineForBranch defines debian upstream version for branches.
func defineForBranch(controlDefaultVersion, branch string) (string, error) {
	switch branch {
	case "ferretdb":
		return fmt.Sprintf("%s~%s", controlDefaultVersion, branch), nil
	default:
		return "", fmt.Errorf("unhandled branch %q", branch)
	}
}

// defineForTag defines debian upstream version for release builds by
// extracting major, minor, patch.
//
// If the tag contains more information such as `v0.100.0-ferretdb-2.0.1`,
// it replaces disallowed characters with `~` returning `0.100.0~ferretdb~2.0.1`.
func defineForTag(tag string) (string, error) {
	match := tagVer.FindStringSubmatch(tag)
	if match == nil || len(match) != tagVer.NumSubexp()+1 {
		return "", fmt.Errorf("unexpected tag syntax %q", tag)
	}

	major := match[tagVer.SubexpIndex("major")]
	minor := match[tagVer.SubexpIndex("minor")]
	patch := match[tagVer.SubexpIndex("patch")]

	semVer := fmt.Sprintf("%s.%s.%s", major, minor, patch)

	rest := match[tagVer.SubexpIndex("rest")]
	if rest == "" {
		return semVer, nil
	}

	rest = disallowedVer.ReplaceAllString(rest, "~")

	return fmt.Sprintf("%s~%s", semVer, rest), nil
}

// setResults sets action output parameters, summary, etc.
func setResults(action *githubactions.Action, res string) {
	output := fmt.Sprintf("version: %s", res)

	action.AddStepSummary(output)
	action.Infof("%s", output)
	action.SetOutput("version", res)
}
