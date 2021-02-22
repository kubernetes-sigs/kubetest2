/*
Copyright 2021 The Kubernetes Authors.

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
package build

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"k8s.io/klog"

	"sigs.k8s.io/kubetest2/pkg/build"
	"sigs.k8s.io/kubetest2/pkg/exec"
)

type gkeBuildAction string

const (
	compile      gkeBuildAction = "compile"
	pack         gkeBuildAction = "package"
	validate     gkeBuildAction = "validate"
	stage        gkeBuildAction = "push-gcs"
	printVersion gkeBuildAction = "print-version"
)

const (
	// GKEMakeStrategy builds and stages using the gke_make build
	GKEMakeStrategy build.BuildAndStageStrategy = "gke_make"
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

var _ build.Builder = &GKEMake{}

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

var _ build.Stager = &GKEMake{}
