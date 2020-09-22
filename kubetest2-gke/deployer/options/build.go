package options

import (
	"fmt"

	"sigs.k8s.io/kubetest2/pkg/build"
)

type BuildOptions struct {
	Strategy      string `flag:"~strategy" desc:"Determines the build strategy to use either make or bazel."`
	StageLocation string `flag:"~stage" desc:"Upload binaries to gs://bucket/ci/job-suffix if set"`
	RepoRoot      string `flag:"-"`
	build.Builder
	build.Stager
}

func (bo *BuildOptions) Validate() error {
	return bo.implementationFromStrategy()
}

func (bo *BuildOptions) implementationFromStrategy() error {
	switch bo.Strategy {
	case "bazel":
		bazel := &build.Bazel{
			RepoRoot:      bo.RepoRoot,
			StageLocation: bo.StageLocation,
			ImageLocation: "gke.gcr.io",
		}
		bo.Builder = bazel
		bo.Stager = bazel
	case "make":
		bo.Builder = &build.MakeBuilder{}
		bo.Stager = &build.ReleasePushBuild{
			Location: bo.StageLocation,
		}
	default:
		return fmt.Errorf("unknown build strategy: %v", bo.Strategy)
	}
	return nil
}
