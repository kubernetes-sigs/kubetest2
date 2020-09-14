package options

import (
	"fmt"
	"regexp"

	"github.com/google/uuid"

	"sigs.k8s.io/kubetest2/pkg/build"
)

type BuildOptions struct {
	Strategy      string `flag:"~strategy" desc:"Determines the build strategy to use either make or bazel."`
	StageLocation string `flag:"~stage" desc:"Upload binaries to gs://bucket/ci/job-suffix if set"`
	Version       string `flag:"~version" desc:"Use a specific GKE version e.g. 1.16.13.gke-400 or 'latest' or 'source' which will build kubernetes from source."`
	RepoRoot      string `flag:"~repo-root" desc:"Path to root of the kubernetes repo. Used only with version=source."`

	build.Builder
	build.Stager
}

func (bo *BuildOptions) Validate() error {
	if err := validateVersion(bo.Version); err != nil {
		return err
	}
	if bo.Version == "source" {
		if bo.RepoRoot == "" {
			return fmt.Errorf("required repo-root when building from source")
		}
		return bo.implementationFromStrategy()
	}
	return nil
}

func (bo *BuildOptions) implementationFromStrategy() error {
	switch bo.Strategy {
	case "bazel":
		// TODO(amwat): find a stable tag
		// The prefix needs to be one of the supported version
		bo.Version = "1.16.0-gke.0.0+" + uuid.New().String()
		bazel := &build.Bazel{
			RepoRoot:      bo.RepoRoot,
			StageLocation: bo.StageLocation,
			ImageLocation: "gke.gcr.io",
			Version:       bo.Version,
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

func validateVersion(version string) error {
	switch version {
	case "latest":
		return nil
	case "source":
		return nil
	default:
		re, err := regexp.Compile(`(\d)\.(\d)+\.(\d)*(-gke\.\d*\.\d*)(.*)`)
		if err != nil {
			return err
		}
		if !re.MatchString(version) {
			return fmt.Errorf("unknown version %q", version)
		}
	}
	return nil
}
