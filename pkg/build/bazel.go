package build

import (
	"os"

	"k8s.io/klog"

	"sigs.k8s.io/kubetest2/pkg/exec"
)

type Bazel struct {
	RepoRoot      string
	StageLocation string
	ImageLocation string
	Version       string
}

var _ Builder = &Bazel{}
var _ Stager = &Bazel{}

const (
	defaultImageRegistry = "k8s.gcr.io"
)

func (b *Bazel) Stage() error {
	if b.ImageLocation == "" {
		b.ImageLocation = defaultImageRegistry
	}
	location := b.StageLocation + "/v" + b.Version
	klog.V(0).Infof("Staging builds to %s ...", location)
	cmd := exec.Command("bazel", "run", "//:push-build", "--", location)
	env := os.Environ()
	env = append(env, "KUBE_DOCKER_REGISTRY="+b.ImageLocation)
	cmd.SetDir(b.RepoRoot)
	cmd.SetEnv(env...)
	exec.InheritOutput(cmd)
	return cmd.Run()
}

func (b *Bazel) Build() error {
	klog.V(0).Infof("Building kubernetes from %s ...", b.RepoRoot)
	cmd := exec.Command("bazel", "build", "//build/release-tars")
	cmd = cmd.SetDir(b.RepoRoot)
	exec.InheritOutput(cmd)
	return cmd.Run()
}
