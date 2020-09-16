package build

import (
	"fmt"
	"os"

	"k8s.io/klog"

	"sigs.k8s.io/kubetest2/pkg/exec"
)

type Bazel struct {
	RepoRoot      string
	StageLocation string
	ImageLocation string
}

var _ Builder = &Bazel{}
var _ Stager = &Bazel{}

const (
	defaultImageRegistry = "k8s.gcr.io"
)

func (b *Bazel) Stage(version string) error {
	if b.ImageLocation == "" {
		b.ImageLocation = defaultImageRegistry
	}
	location := b.StageLocation + "/v" + version
	klog.V(0).Infof("Staging builds to %s ...", location)
	cmd := exec.Command("bazel", "run", "//:push-build", "--", location)
	env := os.Environ()
	env = append(env, "KUBE_DOCKER_REGISTRY="+b.ImageLocation)
	cmd.SetDir(b.RepoRoot)
	cmd.SetEnv(env...)
	exec.InheritOutput(cmd)
	return cmd.Run()
}

func (b *Bazel) Build() (string, error) {
	klog.V(0).Infof("Building kubernetes from %s ...", b.RepoRoot)
	version, err := sourceVersion(b.RepoRoot)
	if err != nil {
		return "", fmt.Errorf("failed to get version: %v", err)
	}
	cmd := exec.Command("bazel", "build", "//build/release-tars")
	cmd = cmd.SetDir(b.RepoRoot)
	exec.InheritOutput(cmd)
	return version, cmd.Run()
}
