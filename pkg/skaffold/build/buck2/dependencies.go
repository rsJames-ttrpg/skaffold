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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

const sourceQuery = "inputs(deps('%s'))"

func query(target string) string {
	return fmt.Sprintf(sourceQuery, target)
}

var once sync.Once

var projectFileCandidates = []string{".buckconfig"}

// GetDependencies finds the source dependencies for the given buck2 artifact.
// All paths are relative to the workspace.
func GetDependencies(ctx context.Context, dir string, a *latest.Buck2Artifact) ([]string, error) {
	timer := time.NewTimer(1 * time.Second)
	defer timer.Stop()

	go func() {
		<-timer.C
		once.Do(func() { log.Entry(ctx).Warn("Retrieving Buck2 dependencies can take a long time the first time") })
	}()

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("unable to find absolute path for %q: %w", dir, err)
	}
	absDir, err = filepath.EvalSymlinks(absDir)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve symlinks in %q: %w", absDir, err)
	}

	projectDir, projectFiles, err := findProject(absDir)
	if err != nil {
		return nil, fmt.Errorf("unable to find the .buckconfig file: %w", err)
	}

	cmd := exec.CommandContext(ctx, "buck2", "uquery", query(a.BuildTarget))
	cmd.Dir = dir
	stdout, err := util.RunCmdOut(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("getting buck2 dependencies: %w", err)
	}

	labels := strings.Split(string(stdout), "\n")
	var deps []string
	for _, l := range labels {
		if l == "" {
			continue
		}
		// Skip external dependencies
		if strings.HasPrefix(l, "@") {
			continue
		}

		rel, err := filepath.Rel(absDir, filepath.Join(projectDir, depToPath(l)))
		if err != nil {
			return nil, fmt.Errorf("unable to find absolute path: %w", err)
		}
		deps = append(deps, rel)
	}

	for _, projectFile := range projectFiles {
		rel, err := filepath.Rel(absDir, filepath.Join(projectDir, projectFile))
		if err != nil {
			return nil, fmt.Errorf("unable to find absolute path: %w", err)
		}
		deps = append(deps, rel)
	}

	log.Entry(ctx).Debugf("Found dependencies for buck2 artifact: %v", deps)

	return deps, nil
}

func depToPath(dep string) string {
	return strings.TrimPrefix(strings.Replace(strings.TrimPrefix(dep, "//"), ":", "/", 1), "/")
}

func findProject(workingDir string) (string, []string, error) {
	dir := workingDir
	for {
		var projectFiles []string
		for _, candidate := range projectFileCandidates {
			if _, err := os.Stat(filepath.Join(dir, candidate)); err == nil {
				projectFiles = append(projectFiles, candidate)
			}
		}
		if len(projectFiles) > 0 {
			return dir, projectFiles, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil, fmt.Errorf("unable to find .buckconfig in %q or any parent directory", workingDir)
		}
		dir = parent
	}
}
