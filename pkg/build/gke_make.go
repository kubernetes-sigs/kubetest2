package build

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"k8s.io/klog"

	"sigs.k8s.io/kubetest2/pkg/exec"
)

type gkeBuildAction string

const (
	compile      gkeBuildAction = "compile"
	pack                        = "package"
	validate                    = "validate"
	stage                       = "push-gcs"
	printVersion                = "print-version"
)

type GKEMake struct {
	RepoRoot      string
	BuildScript   string
	VersionSuffix string
	StageLocation string
}

func gkeBuildActions(actions []gkeBuildAction) string {
	stringActions := make([]string, len(actions))
	for i, action := range actions {
		stringActions[i] = string(action)
	}
	return strings.Join(stringActions, ",")
}

func arg(key, value string) string {
	return fmt.Sprintf("%s=%s", key, value)
}

func (gmb *GKEMake) runWithActions(stdout, stderr io.Writer, actions []gkeBuildAction, extraArgs ...string) error {

	args := []string{arg("GKE_BUILD_ACTIONS", gkeBuildActions(actions))}
	args = append(args, extraArgs...)
	cmd := exec.Command(gmb.BuildScript, args...)
	cmd.SetDir(gmb.RepoRoot)
	cmd.SetStdout(stdout)
	cmd.SetStderr(stderr)
	return cmd.Run()
}

func (gmb *GKEMake) Build() (string, error) {
	klog.V(2).Infof("starting gke build ...")
	if err := gmb.runWithActions(os.Stdout, os.Stderr, []gkeBuildAction{compile, validate}, arg("VERSION_SUFFIX", gmb.VersionSuffix)); err != nil {
		return "", err
	}
	version := &bytes.Buffer{}
	if err := gmb.runWithActions(version, ioutil.Discard, []gkeBuildAction{printVersion}); err != nil {
		return "", err
	}
	return version.String(), nil
}

var _ Builder = &GKEMake{}

func (gmb *GKEMake) Stage(version string) error {
	klog.V(2).Infof("staging gke builds ...")
	args := []string{arg("VERSION", version)}
	if gmb.StageLocation != "" {
		args = append(args, arg("GCS_BUCKET", gmb.StageLocation))
	}

	if err := gmb.runWithActions(os.Stdout, os.Stderr, []gkeBuildAction{pack, stage}, args...); err != nil {
		return err
	}
	return nil
}

var _ Stager = &GKEMake{}
