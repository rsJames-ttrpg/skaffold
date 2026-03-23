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
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/google/go-containerregistry/pkg/v1/tarball"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

// Build builds an artifact with Buck2.
func (b *Builder) Build(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string, matcher platform.Matcher) (string, error) {
	if matcher.IsMultiPlatform() {
		log.Entry(ctx).Warnf("multiple target platforms %q found for artifact %q. Skaffold doesn't yet support multi-platform builds for the buck2 builder. Consider specifying a single target platform explicitly.", matcher.String(), artifact.ImageName)
	}

	a := artifact.ArtifactType.Buck2Artifact

	tarPath, err := b.buildTar(ctx, out, artifact.Workspace, a, matcher)
	if err != nil {
		return "", err
	}

	if b.pushImages {
		return docker.Push(tarPath, tag, b.cfg, nil)
	}
	return b.loadImage(ctx, out, tarPath, tag)
}

func (b *Builder) SupportedPlatforms() platform.Matcher { return platform.All }

func (b *Builder) buildTar(ctx context.Context, out io.Writer, workspace string, a *latest.Buck2Artifact, matcher platform.Matcher) (string, error) {
	if !strings.HasSuffix(a.BuildTarget, ".tar") {
		return "", errors.New("the buck2 build target should end with .tar")
	}

	args := []string{"build"}
	args = append(args, a.BuildArgs...)
	args = append(args, a.BuildTarget)

	for _, mapping := range a.PlatformMappings {
		m, err := platform.Parse([]string{mapping.Platform})
		if err == nil {
			if matcher.Intersect(m).IsNotEmpty() {
				args = append(args, fmt.Sprintf("--target-platforms=%s", mapping.Buck2PlatformTarget))
			}
		}
	}

	cmd := exec.CommandContext(ctx, "buck2", args...)
	cmd.Dir = workspace
	cmd.Stdout = out
	cmd.Stderr = out
	if err := util.RunCmd(ctx, cmd); err != nil {
		return "", fmt.Errorf("running command: %w", err)
	}

	tarPath, err := buck2TarPath(ctx, workspace, a)
	if err != nil {
		return "", fmt.Errorf("getting buck2 tar path: %w", err)
	}

	return tarPath, nil
}

func (b *Builder) loadImage(ctx context.Context, out io.Writer, tarPath string, tag string) (string, error) {
	manifest, err := tarball.LoadManifest(func() (io.ReadCloser, error) {
		return os.Open(tarPath)
	})

	if err != nil {
		return "", fmt.Errorf("loading manifest from tarball failed: %w", err)
	}

	imageTar, err := os.Open(tarPath)
	if err != nil {
		return "", fmt.Errorf("opening image tarball: %w", err)
	}
	defer imageTar.Close()

	buck2Tag := manifest[0].RepoTags[0]
	imageID, err := b.localDocker.Load(ctx, out, imageTar, buck2Tag)
	if err != nil {
		return "", fmt.Errorf("loading image into docker daemon: %w", err)
	}

	if err := b.localDocker.Tag(ctx, imageID, tag); err != nil {
		return "", fmt.Errorf("tagging the image: %w", err)
	}

	return imageID, nil
}

func buck2TarPath(ctx context.Context, workspace string, a *latest.Buck2Artifact) (string, error) {
	args := []string{"build", "--show-full-output"}
	args = append(args, a.BuildArgs...)
	args = append(args, a.BuildTarget)

	cmd := exec.CommandContext(ctx, "buck2", args...)
	cmd.Dir = workspace

	buf, err := util.RunCmdOut(ctx, cmd)
	if err != nil {
		return "", err
	}

	// buck2 build --show-full-output prints lines like:
	// <target> <output_path>
	output := strings.TrimSpace(string(buf))
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			return parts[len(parts)-1], nil
		}
	}

	return "", fmt.Errorf("unable to determine output path from buck2 build output: %s", output)
}
