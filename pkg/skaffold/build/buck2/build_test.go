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
	"io"
	"testing"

	specs "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestBuildBuck2(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.NewTempDir().Mkdir("bin").Chdir()
		t.Override(&util.DefaultExecCommand, testutil.CmdRun("buck2 build //:app.tar").AndRunOut(
			"buck2 build --show-full-output //:app.tar",
			"//:app.tar bin/app.tar"))
		testutil.CreateFakeImageTar("buck2:app", "bin/app.tar")

		artifact := &latest.Artifact{
			Workspace: ".",
			ArtifactType: latest.ArtifactType{
				Buck2Artifact: &latest.Buck2Artifact{
					BuildTarget: "//:app.tar",
				},
			},
		}

		builder := NewArtifactBuilder(fakeLocalDaemon(), &mockConfig{}, false)
		_, err := builder.Build(context.Background(), io.Discard, artifact, "img:tag", platform.Matcher{})

		t.CheckNoError(err)
	})
}

func TestBuildBuck2WithPlatforms(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.NewTempDir().Mkdir("bin").Chdir()
		t.Override(&util.DefaultExecCommand, testutil.CmdRun("buck2 build //:app.tar --target-platforms=//platforms:linux-x86_64").AndRunOut(
			"buck2 build --show-full-output //:app.tar",
			"//:app.tar bin/app.tar"))
		testutil.CreateFakeImageTar("buck2:app", "bin/app.tar")

		artifact := &latest.Artifact{
			Workspace: ".",
			ArtifactType: latest.ArtifactType{
				Buck2Artifact: &latest.Buck2Artifact{
					BuildTarget: "//:app.tar",
					PlatformMappings: []latest.Buck2PlatformMapping{
						{
							Platform:            "linux/amd64",
							Buck2PlatformTarget: "//platforms:linux-x86_64",
						},
					},
				},
			},
		}

		testPlatform := platform.Matcher{Platforms: []specs.Platform{{Architecture: "amd64", OS: "linux"}}}

		builder := NewArtifactBuilder(fakeLocalDaemon(), &mockConfig{}, false)
		_, err := builder.Build(context.Background(), io.Discard, artifact, "img:tag", testPlatform)

		t.CheckNoError(err)
	})
}

func TestBuildBuck2FailInvalidTarget(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		artifact := &latest.Artifact{
			ArtifactType: latest.ArtifactType{
				Buck2Artifact: &latest.Buck2Artifact{
					BuildTarget: "//:invalid-target",
				},
			},
		}

		builder := NewArtifactBuilder(nil, &mockConfig{}, false)
		_, err := builder.Build(context.Background(), io.Discard, artifact, "img:tag", platform.Matcher{})

		t.CheckErrorContains("the buck2 build target should end with .tar", err)
	})
}

func TestBuck2TarPath(t *testing.T) {
	testutil.Run(t, "basic", func(t *testutil.T) {
		t.Override(&util.DefaultExecCommand, testutil.CmdRunOut(
			"buck2 build --show-full-output //:skaffold_example.tar",
			"//:skaffold_example.tar buck-out/gen/skaffold_example.tar\n",
		))

		tarPath, err := buck2TarPath(context.Background(), ".", &latest.Buck2Artifact{
			BuildTarget: "//:skaffold_example.tar",
		})

		t.CheckNoError(err)
		t.CheckDeepEqual("buck-out/gen/skaffold_example.tar", tarPath)
	})

	testutil.Run(t, "with args", func(t *testutil.T) {
		t.Override(&util.DefaultExecCommand, testutil.CmdRunOut(
			"buck2 build --show-full-output --arg1 --arg2 //:skaffold_example.tar",
			"//:skaffold_example.tar buck-out/gen/skaffold_example.tar\n",
		))

		tarPath, err := buck2TarPath(context.Background(), ".", &latest.Buck2Artifact{
			BuildArgs:   []string{"--arg1", "--arg2"},
			BuildTarget: "//:skaffold_example.tar",
		})

		t.CheckNoError(err)
		t.CheckDeepEqual("buck-out/gen/skaffold_example.tar", tarPath)
	})
}

func fakeLocalDaemon() docker.LocalDaemon {
	return docker.NewLocalDaemon(&testutil.FakeAPIClient{}, nil, false, nil)
}

type mockConfig struct {
	docker.Config
}

func (c *mockConfig) GetInsecureRegistries() map[string]bool { return nil }
