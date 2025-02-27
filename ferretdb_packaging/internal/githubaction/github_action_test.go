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

package githubaction

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSemVar(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		tag        string
		major      string
		minor      string
		patch      string
		prerelease string
		err        string
	}{
		"Valid": {
			tag:        "v1.100.0-ferretdb",
			major:      "1",
			minor:      "100",
			patch:      "0",
			prerelease: "ferretdb",
		},
		"SpecificVersion": {
			tag:        "v1.100.0-ferretdb-2.0.1",
			major:      "1",
			minor:      "100",
			patch:      "0",
			prerelease: "ferretdb-2.0.1",
		},
		"MissingV": {
			tag: "0.100.0-ferretdb",
			err: `unexpected tag syntax "0.100.0-ferretdb"`,
		},
		"MissingFerretDB": {
			tag: "v0.100.0",
			err: "prerelease is empty",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			major, minor, patch, prerelease, err := SemVar(tc.tag)

			if tc.err != "" {
				require.EqualError(t, err, tc.err)
				return
			}

			require.Equal(t, tc.major, major)
			require.Equal(t, tc.minor, minor)
			require.Equal(t, tc.patch, patch)
			require.Equal(t, tc.prerelease, prerelease)
		})
	}
}
