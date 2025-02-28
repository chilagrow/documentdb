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
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/sethvargo/go-githubactions"
)

// semVerTag is a https://semver.org/#is-there-a-suggested-regular-expression-regex-to-check-a-semver-string,
// but with a leading `v`.
var semVerTag = regexp.MustCompile(`^v(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)(?:-(?P<prerelease>(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+(?P<buildmetadata>[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

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

// semVar parses tag and returns version components.
//
// It returns error for invalid tag syntax, prerelease is missing `ferretdb` or if it has buildmetadata.
func semVar(tag string) (major, minor, patch, prerelease string, err error) {
	match := semVerTag.FindStringSubmatch(tag)
	if match == nil || len(match) != semVerTag.NumSubexp()+1 {
		err = fmt.Errorf("unexpected tag syntax %q", tag)
		return
	}

	major = match[semVerTag.SubexpIndex("major")]
	minor = match[semVerTag.SubexpIndex("minor")]
	patch = match[semVerTag.SubexpIndex("patch")]
	prerelease = match[semVerTag.SubexpIndex("prerelease")]
	buildmetadata := match[semVerTag.SubexpIndex("buildmetadata")]

	if prerelease == "" {
		err = fmt.Errorf("prerelease is empty")
		return
	}

	if !strings.Contains(prerelease, "ferretdb") {
		err = fmt.Errorf("prerelease %q should include `ferretdb`", prerelease)
		return
	}

	if buildmetadata != "" {
		err = fmt.Errorf("buildmetadata %q is present", buildmetadata)
		return
	}

	return
}
