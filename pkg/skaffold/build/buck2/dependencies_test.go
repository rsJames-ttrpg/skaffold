/*
Copyright 2024 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package buck2

import (
	"context"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestGetDependencies(t *testing.T) {
	tests := []struct {
		description   string
		workspace     string
		target        string
		files         map[string]string
		expectedQuery string
		output        string
		expected      []string
		shouldErr     bool
	}{
		{
			description: "with .buckconfig",
			workspace:   ".",
			target:      "target",
			files: map[string]string{
				".buckconfig": "",
				"BUCK":        "",
				"dep1":        "",
				"dep2":        "",
			},
			expectedQuery: "buck2 uquery inputs(deps('target'))",
			output:        "@ignored\n//:BUCK\n\n//:dep1\n//:dep2\n",
			expected:      []string{"BUCK", "dep1", "dep2", ".buckconfig"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir()
			t.Override(&util.DefaultExecCommand, testutil.CmdRunOut(
				test.expectedQuery,
				test.output,
			))
			tmpDir.WriteFiles(test.files).Chdir()

			deps, err := GetDependencies(context.Background(), test.workspace, &latest.Buck2Artifact{
				BuildTarget: test.target,
			})

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, deps)
		})
	}
}

func TestGetDependenciesWithNoProject(t *testing.T) {
	testutil.Run(t, "without .buckconfig", func(t *testutil.T) {
		tmpDir := t.NewTempDir()

		_, err := GetDependencies(context.Background(), tmpDir.Root(), &latest.Buck2Artifact{
			BuildTarget: "target",
		})

		shouldErr := true
		t.CheckError(shouldErr, err)
	})
}

func TestQuery(t *testing.T) {
	q := query("//:skaffold_example.tar")

	expectedQuery := `inputs(deps('//:skaffold_example.tar'))`
	if q != expectedQuery {
		t.Errorf("Expected [%s]. Got [%s]", expectedQuery, q)
	}
}

func TestDepToPath(t *testing.T) {
	tests := []struct {
		description string
		dep         string
		expected    string
	}{
		{
			description: "top level file",
			dep:         "//:dispatcher.go",
			expected:    "dispatcher.go",
		},
		{
			description: "nested file",
			dep:         "//vendor/github.com/gorilla/mux:mux.go",
			expected:    "vendor/github.com/gorilla/mux/mux.go",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			path := depToPath(test.dep)

			t.CheckDeepEqual(test.expected, path)
		})
	}
}
