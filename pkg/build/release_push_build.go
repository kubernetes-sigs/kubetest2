package build

import (
	"fmt"
	"regexp"

	"sigs.k8s.io/kubetest2/pkg/exec"
)

type ReleasePushBuild struct {
	Location string
}

var _ Stager = &ReleasePushBuild{}

// Stage stages the build to GCS using
// essentially release/push-build.sh --bucket=B --ci --gcs-suffix=S --noupdatelatest
func (rpb *ReleasePushBuild) Stage() error {
	re := regexp.MustCompile(`^gs://([\w-]+)/(devel|ci)(/.*)?`)
	mat := re.FindStringSubmatch(rpb.Location)
	if mat == nil {
		return fmt.Errorf("invalid stage location: %v. Use gs://bucket/ci/optional-suffix", rpb.Location)
	}
	bucket := mat[1]
	ci := mat[2] == "ci"
	gcsSuffix := mat[3]

	args := []string{
		"--nomock",
		"--verbose",
		"--noupdatelatest",
		fmt.Sprintf("--bucket=%v", bucket),
	}
	if len(gcsSuffix) > 0 {
		args = append(args, fmt.Sprintf("--gcs-suffix=%v", gcsSuffix))
	}
	if ci {
		args = append(args, "--ci")
	}

	name, err := K8sDir("release", "push-build.sh")
	if err != nil {
		return err
	}
	cmd := exec.Command(name, args...)
	exec.InheritOutput(cmd)
	cmdDir, err := K8sDir("kubernetes")
	if err != nil {
		return err
	}
	cmd.SetDir(cmdDir)
	return cmd.Run()
}
