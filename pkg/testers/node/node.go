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

// Package node implements a node tester that implements e2e node testing following
// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-node/e2e-node-tests.md#delete-instance-after-tests-run
// https://github.com/kubernetes/kubernetes/blob/96be00df69390ed41b8ec22facc43bcbb9c88aae/build/root/Makefile#L206-L271
// currently only support REMOTE=true
package node

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/octago/sflags/gen/gpflag"
	"k8s.io/klog"
	"sigs.k8s.io/boskos/client"

	"sigs.k8s.io/kubetest2/pkg/boskos"
	"sigs.k8s.io/kubetest2/pkg/exec"
)

const (
	target                 = "test-e2e-node"
	gceProjectResourceType = "gce-project"
)

type Tester struct {
	RepoRoot                    string `desc:"Absolute path to kubernetes repository root."`
	GCPProject                  string `desc:"GCP Project to create VMs in. If unset, the deployer will attempt to get a project from boskos."`
	GCPZone                     string `desc:"GCP Zone to create VMs in."`
	SkipRegex                   string `desc:"Regular expression of jobs to skip."`
	FocusRegex                  string `desc:"Regular expression of jobs to focus on."`
	Runtime                     string `desc:"Container runtime to use."`
	TestArgs                    string `desc:"A space-separated list of arguments to pass to node e2e test."`
	BoskosAcquireTimeoutSeconds int    `desc:"How long (in seconds) to hang on a request to Boskos to acquire a resource before erroring."`
	BoskosLocation              string `desc:"If set, manually specifies the location of the boskos server. If unset and boskos is needed"`

	// boskos struct field will be non-nil when the deployer is
	// using boskos to acquire a GCP project
	boskos *client.Client

	// this channel serves as a signal channel for the hearbeat goroutine
	// so that it can be explicitly closed
	boskosHeartbeatClose chan struct{}
}

func NewDefaultTester() *Tester {
	return &Tester{
		SkipRegex:                   `\[Flaky\]|\[Slow\]|\[Serial\]`,
		Runtime:                     "docker",
		BoskosLocation:              "http://boskos.test-pods.svc.cluster.local.",
		BoskosAcquireTimeoutSeconds: 5 * 60,
	}
}

func (t *Tester) Execute() error {
	fs, err := gpflag.Parse(t)
	if err != nil {
		return fmt.Errorf("failed to initialize tester: %v", err)
	}

	klog.InitFlags(nil)
	fs.AddGoFlagSet(flag.CommandLine)

	help := fs.BoolP("help", "h", false, "")
	if err := fs.Parse(os.Args); err != nil {
		return fmt.Errorf("failed to parse flags: %v", err)
	}

	if *help {
		fs.SetOutput(os.Stdout)
		fs.PrintDefaults()
		return nil
	}
	if err := t.validateFlags(); err != nil {
		return fmt.Errorf("failed to validate flags: %v", err)
	}

	// try to acquire project from boskos
	if t.GCPProject == "" {
		klog.V(1).Info("no GCP project provided, acquiring from Boskos ...")

		boskosClient, err := boskos.NewClient(t.BoskosLocation)
		if err != nil {
			return fmt.Errorf("failed to make boskos client: %s", err)
		}
		t.boskos = boskosClient

		resource, err := boskos.Acquire(
			t.boskos,
			gceProjectResourceType,
			time.Duration(t.BoskosAcquireTimeoutSeconds)*time.Second,
			t.boskosHeartbeatClose,
		)

		if err != nil {
			return fmt.Errorf("init failed to get project from boskos: %s", err)
		}
		t.GCPProject = resource.Name
		klog.V(1).Infof("got project %s from boskos", t.GCPProject)
	}

	defer func() {
		if t.boskos != nil {
			klog.V(1).Info("releasing boskos project")
			err := boskos.Release(
				t.boskos,
				t.GCPProject,
				t.boskosHeartbeatClose,
			)
			if err != nil {
				klog.Errorf("failed to release boskos project: %v", err)
			}
		}
	}()

	return t.Test()
}

func (t *Tester) validateFlags() error {
	if t.RepoRoot == "" {
		return fmt.Errorf("required --repo-root")
	}
	if t.GCPZone == "" {
		return fmt.Errorf("required --gcp-zone")
	}
	return nil
}

func (t *Tester) constructArgs() []string {
	defaultArgs := []string{
		"REMOTE=true",
		"DELETE_INSTANCES=true",
	}

	argsFromFlags := []string{
		"SKIP=" + t.SkipRegex,
		"FOCUS=" + t.FocusRegex,
		"RUNTIME=" + t.Runtime,
		// https://github.com/kubernetes/kubernetes/blob/96be00df69390ed41b8ec22facc43bcbb9c88aae/hack/make-rules/test-e2e-node.sh#L120
		// TODO: this should be configurable without overriding at the gcloud env level
		"CLOUDSDK_CORE_PROJECT=" + t.GCPProject,
		// https://github.com/kubernetes/kubernetes/blob/96be00df69390ed41b8ec22facc43bcbb9c88aae/hack/make-rules/test-e2e-node.sh#L113
		"ZONE=" + t.GCPZone,
	}
	return append(defaultArgs, argsFromFlags...)
}

func (t *Tester) Test() error {
	var args []string
	args = append(args, target)
	args = append(args, t.constructArgs()...)
	cmd := exec.Command("make", args...)
	cmd.SetDir(t.RepoRoot)
	exec.InheritOutput(cmd)
	return cmd.Run()
}

func Main() {
	t := NewDefaultTester()
	if err := t.Execute(); err != nil {
		klog.Fatalf("failed to run ginkgo tester: %v", err)
	}
}
