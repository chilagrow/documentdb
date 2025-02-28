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
