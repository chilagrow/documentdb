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
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/sethvargo/go-githubactions"
)

func main() {
	commandF := flag.String("command", "", "command to run, possible values: [deb-version, docker-tags]")

	controlFileF := flag.String("control-file", "../pg_documentdb/documentdb.control", "pg_documentdb/documentdb.control file path")

	flag.Parse()

	action := githubactions.New()

	debugEnv(action)

	if *commandF == "" {
		action.Fatalf("-command flag is empty.")
	}

	switch *commandF {
	case "deb-version":
		if *controlFileF == "" {
			action.Fatalf("%s", "-control-file flag is empty.")
		}

		controlDefaultVersion, err := getControlDefaultVersion(*controlFileF)
		if err != nil {
			action.Fatalf("%s", err)
		}

		packageVersion, err := definePackageVersion(controlDefaultVersion, action.Getenv)
		if err != nil {
			action.Fatalf("%s", err)
		}

		setDebianVersionResults(action, packageVersion)
	case "docker-tags":
		res, err := defineDockerTags(action.Getenv)
		if err != nil {
			action.Fatalf("%s", err)
		}

		setDockerTagsResults(action, res)
	default:
		action.Fatalf("unhandled command %q", *commandF)
	}
}

// controlDefaultVer matches major, minor and "patch" from default_version field in control file,
// see pg_documentdb_core/documentdb_core.control.
var controlDefaultVer = regexp.MustCompile(`(?m)^default_version = '(?P<major>[0-9]+)\.(?P<minor>[0-9]+)-(?P<patch>[0-9]+)'$`)

// disallowedVer matches disallowed characters of Debian `upstream_version` when used without `debian_revision`.
// See https://www.debian.org/doc/debian-policy/ch-controlfields.html#version.
var disallowedVer = regexp.MustCompile(`[^A-Za-z0-9~.+]`)

// getControlDefaultVersion returns the default_version field from the control file
// in SemVer format (0.100-0 -> 0.100.0).
func getControlDefaultVersion(f string) (string, error) {
	b, err := os.ReadFile(f)
	if err != nil {
		return "", err
	}

	match := controlDefaultVer.FindSubmatch(b)
	if match == nil || len(match) != controlDefaultVer.NumSubexp()+1 {
		return "", fmt.Errorf("control file did not find default_version: %s", f)
	}

	major := match[controlDefaultVer.SubexpIndex("major")]
	minor := match[controlDefaultVer.SubexpIndex("minor")]
	patch := match[controlDefaultVer.SubexpIndex("patch")]

	return fmt.Sprintf("%s.%s.%s", major, minor, patch), nil
}

// definePackageVersion returns valid Debian package version,
// based on `default_version` in the control file and environment variables set by GitHub Actions.
//
// See https://www.debian.org/doc/debian-policy/ch-controlfields.html#version.
// We use `upstream_version` only.
// For that reason, we can't use `-`, so we replace it with `~`.
func definePackageVersion(controlDefaultVersion string, getenv githubactions.GetenvFunc) (string, error) {
	var packageVersion string
	var err error

	switch event := getenv("GITHUB_EVENT_NAME"); event {
	case "pull_request", "pull_request_target":
		branch := strings.ToLower(getenv("GITHUB_HEAD_REF"))
		packageVersion = definePackageVersionForPR(controlDefaultVersion, branch)

	case "push", "schedule", "workflow_run":
		refName := strings.ToLower(getenv("GITHUB_REF_NAME"))

		switch refType := strings.ToLower(getenv("GITHUB_REF_TYPE")); refType {
		case "branch":
			packageVersion, err = definePackageVersionForBranch(controlDefaultVersion, refName)

		case "tag":
			packageVersion, err = definePackagerVersionForTag(refName)

		default:
			err = fmt.Errorf("unhandled ref type %q for event %q", refType, event)
		}

	default:
		err = fmt.Errorf("unhandled event type %q", event)
	}

	if err != nil {
		return "", err
	}

	if packageVersion == "" {
		return "", fmt.Errorf("both packageVersion and err are nil")
	}

	return packageVersion, nil
}

// definePackageVersionForPR returns valid Debian package version for PR.
// See [definePackageVersion].
func definePackageVersionForPR(controlDefaultVersion, branch string) string {
	// for branches like "dependabot/submodules/XXX"
	parts := strings.Split(branch, "/")
	branch = parts[len(parts)-1]
	res := fmt.Sprintf("%s-pr-%s", controlDefaultVersion, branch)

	return disallowedVer.ReplaceAllString(res, "~")
}

// definePackageVersionForBranch returns valid Debian package version for branch.
// See [definePackageVersion].
func definePackageVersionForBranch(controlDefaultVersion, branch string) (string, error) {
	switch branch {
	case "ferretdb":
		return fmt.Sprintf("%s~branch~%s", controlDefaultVersion, branch), nil
	default:
		return "", fmt.Errorf("unhandled branch %q", branch)
	}
}

// definePackagerVersionForTag returns valid Debian package version for tag.
// See [definePackageVersion].
func definePackagerVersionForTag(tag string) (string, error) {
	major, minor, patch, prerelease, err := semVar(tag)
	if err != nil {
		return "", err
	}

	res := fmt.Sprintf("%s.%s.%s-%s", major, minor, patch, prerelease)
	return disallowedVer.ReplaceAllString(res, "~"), nil
}

// setDebianVersionResults sets action output parameters, summary, etc.
func setDebianVersionResults(action *githubactions.Action, res string) {
	output := fmt.Sprintf("version: `%s`", res)

	action.AddStepSummary(output)
	action.Infof("%s", output)
	action.SetOutput("version", res)
}
