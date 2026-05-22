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
	"errors"
	"flag"
	"fmt"
	"os"
	stdexec "os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/octago/sflags/gen/gpflag"
	"k8s.io/klog/v2"

	"sigs.k8s.io/boskos/client"
	"sigs.k8s.io/kubetest2/pkg/artifacts"
	"sigs.k8s.io/kubetest2/pkg/boskos"
	"sigs.k8s.io/kubetest2/pkg/exec"
	"sigs.k8s.io/kubetest2/pkg/fs"
	"sigs.k8s.io/kubetest2/pkg/testers"
)

var GitTag string

const (
	ciPrivateKeyEnv = "GCE_SSH_PRIVATE_KEY_FILE"
	ciPublicKeyEnv  = "GCE_SSH_PUBLIC_KEY_FILE"
)

type Tester struct {
	RepoRoot                       string        `desc:"Absolute path to the kubernetes or provider-aws-test-infra repository root. Only needed when not using some kind of pre-built binaries."`
	UseBuiltBinaries               bool          `desc:"Look for binaries in _rundir/$KUBETEST2_RUN_DIR."`
	UseBinariesFromPath            bool          `desc:"Look for binaries in the $PATH."`
	UseBinariesFromPackage         bool          `desc:"Look for binaries in a kubernetes test package."`
	TestPackageURL                 string        `desc:"The url to download a kubernetes test package from."`
	TestPackageVersion             string        `desc:"The ginkgo tester uses a test package made during the kubernetes build. The tester downloads this test package from one of the release tars published to the Release bucket. Defaults to latest. visit https://kubernetes.io/releases/ to find release names. Example: v1.20.0-alpha.0"`
	TestPackageDir                 string        `desc:"The directory in the bucket which represents the type of release. Default to the release directory."`
	TestPackageMarker              string        `desc:"The version marker in the directory containing the package version to download when unspecified. Defaults to latest.txt."`
	GCPProject                     string        `desc:"GCP Project to create VMs in. If unset, the deployer will attempt to get a project from boskos."`
	GCPZone                        string        `desc:"GCP Zone to create VMs in."`
	SkipRegex                      string        `desc:"Regular expression of jobs to skip."`
	FocusRegex                     string        `desc:"Regular expression of jobs to focus on."`
	TestArgs                       string        `desc:"A space-separated list of arguments to pass to node e2e test."`
	LabelFilter                    string        `desc:"Label filter arguments to be passed to ginkgo."`
	BoskosAcquireTimeoutSeconds    int           `desc:"How long (in seconds) to hang on a request to Boskos to acquire a resource before erroring."`
	BoskosHeartbeatIntervalSeconds int           `desc:"How often (in seconds) to send a heartbeat to Boskos to hold the acquired resource. 0 means no heartbeat."`
	BoskosLocation                 string        `desc:"If set, manually specifies the location of the boskos server. If unset and boskos is needed"`
	ImageConfigFile                string        `desc:"Path to a file containing image configuration."`
	Images                         string        `desc:"List of images to use when creating instances separated by commas"`
	ImageProject                   string        `desc:"A GCP Project containing an image to use when creating instances"`
	InstanceType                   string        `desc:"Machine/Instance type to use on AWS/GCP"`
	InstanceMetadata               string        `desc:"Instance Metadata to use for creating GCE instance"`
	UserDataFile                   string        `desc:"User Data to use for creating EC2 instance"`
	Provider                       string        `desc:"Cloud Provider to use for node tests. Valid options are ec2 and gce"`
	UseDockerizedBuild             bool          `desc:"Use dockerized build for test artifacts"`
	TargetBuildArch                string        `desc:"Target architecture for the test artifacts for dockerized build"`
	ImageConfigDir                 string        `desc:"Path to image config files."`
	Parallelism                    int           `desc:"The number of nodes to run in parallel."`
	GCPProjectType                 string        `desc:"Explicitly indicate which project type to select from boskos."`
	RuntimeConfig                  string        `desc:"The runtime configuration for the API server. Format: a list of key=value pairs."`
	Timeout                        time.Duration `desc:"How long (in golang duration format) to wait for ginkgo tests to complete."`
	DeleteInstances                bool          `desc:"Where to delete instances after running the test"`
	NodeEnv                        string        `desc:"Additional metadata keys to add to a gce instance"`

	// boskos struct field will be non-nil when the deployer is
	// using boskos to acquire a GCP project
	boskos *client.Client

	// this channel serves as a signal channel for the hearbeat goroutine
	// so that it can be explicitly closed
	boskosHeartbeatClose chan struct{}

	// this contains ssh key path
	privateKey string
	sshUser    string

	runDir string

	// These paths are set up while checking for each required binary.
	// The key is the binary base name (e.g. "e2e_node.test").
	paths map[bin]string
}

// These are the binaries required for E2E node testing when using pre-built binaries.
//
// Local and remote OS+arch must be the same and are determined by the platform
// the tester was compiled for. If testing of e.g. linux-arm64 is needed while
// running in a linux-amd64 Prow container, then --repo-root and building for
// the target architecture from source have to be used.
//
// This restriction comes from running the same e2e_node.test binary both
// locally and remotely. It could be removed by acquiring that binary once for
// local and remote platform and adding more parameters to `e2e_node.test
// remote`, but that would be more complicated everywhere.
type bin string

const (
	e2eNodeTest = bin("e2e_node.test")
	ginkgo      = bin("ginkgo")
	kubelet     = bin("kubelet")
)

var (
	testBinaries = []bin{e2eNodeTest, ginkgo, kubelet}
)

func NewDefaultTester() *Tester {
	return &Tester{
		TestPackageURL:                 "https://dl.k8s.io",
		TestPackageDir:                 "release",
		TestPackageMarker:              "latest.txt",
		SkipRegex:                      `\[Flaky\]|\[Slow\]|\[Serial\]`,
		BoskosLocation:                 "http://boskos.test-pods.svc.cluster.local.",
		BoskosAcquireTimeoutSeconds:    5 * 60,
		BoskosHeartbeatIntervalSeconds: 5 * 60,
		Parallelism:                    8,
		boskosHeartbeatClose:           make(chan struct{}),
		GCPProjectType:                 "gce-project",
		Provider:                       "gce",
		DeleteInstances:                true,
		paths:                          make(map[bin]string),
	}
}

func (t *Tester) Execute() error {
	fs, err := gpflag.Parse(t)
	if err != nil {
		return fmt.Errorf("failed to initialize tester: %v", err)
	}

	// initing the klog flags adds them to goflag.CommandLine
	// they can then be added to the built pflag set
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
	if err := t.initKubetest2Info(); err != nil {
		return err
	}
	if err := t.pretestSetup(); err != nil {
		return err
	}
	if err := testers.WriteVersionToMetadata(GitTag, t.TestPackageVersion); err != nil {
		return err
	}

	// Use the KUBE_SSH_USER environment variable if it is set. This is particularly
	// required for Fedora CoreOS hosts that only have the user 'core`. Tests
	// using Fedora CoreOS as a host for node tests must set KUBE_SSH_USER
	// environment variable so that test infrastructure can communicate with the host
	// successfully using ssh.
	if os.Getenv("KUBE_SSH_USER") != "" {
		t.sshUser = os.Getenv("KUBE_SSH_USER")
	} else {
		t.sshUser = os.Getenv("USER")
	}

	if t.Provider == "gce" {
		t.maybeSetupSSHKeys()

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
				t.GCPProjectType,
				time.Duration(t.BoskosAcquireTimeoutSeconds)*time.Second,
				time.Duration(t.BoskosHeartbeatIntervalSeconds)*time.Second,
				t.boskosHeartbeatClose,
			)

			if err != nil {
				return fmt.Errorf("init failed to get project from boskos: %s", err)
			}
			t.GCPProject = resource.Name
			klog.V(1).Infof("got project %s from boskos", t.GCPProject)
		}
	}

	defer func() {
		if t.boskos != nil {
			klog.V(1).Info("releasing boskos project")
			err := boskos.Release(
				t.boskos,
				[]string{t.GCPProject},
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
	modeCount := 0
	if t.RepoRoot != "" {
		modeCount++
	}
	if t.UseBuiltBinaries {
		modeCount++
	}
	if t.UseBinariesFromPath {
		modeCount++
	}
	if t.UseBinariesFromPackage {
		modeCount++
	}
	if modeCount != 1 {
		return errors.New("required exactly one of --repo-root, --use-built-binaries, --use-binaries-from-path, --use-binaries-from-package")
	}
	if t.GCPZone == "" && t.Provider == "gce" {
		return fmt.Errorf("required --gcp-zone")
	}
	return nil
}

// initKubetest2Info sets relevant information from the well defined kubetest2 environment variables.
func (t *Tester) initKubetest2Info() error {
	if dir, ok := os.LookupEnv("KUBETEST2_RUN_DIR"); ok {
		t.runDir = dir
		return nil
	}
	// Binaries can be found in rundir when they are built
	if t.UseBuiltBinaries {
		t.runDir = artifacts.RunDir()
		return nil
	}
	// default to current working directory if for some reason the env is not set
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to set run dir: %v", err)
	}
	t.runDir = dir
	return nil
}

// maybeSetupSSHKeys will best-effort try to setup ssh keys for gcloud to reuse
// from existing files pointed to by "well-known" environment variables used in CI
func (t *Tester) maybeSetupSSHKeys() {
	home, err := os.UserHomeDir()
	if err != nil {
		klog.Warningf("failed to get user's home directory")
		return
	}
	// check if there are existing ssh keys, if either exist don't do anything
	klog.V(2).Info("checking for existing gcloud ssh keys...")
	t.privateKey = filepath.Join(home, ".ssh", "google_compute_engine")
	if _, err := os.Stat(t.privateKey); err == nil {
		klog.V(2).Infof("found existing private key at %s", t.privateKey)
		return
	}
	publicKey := t.privateKey + ".pub"
	if _, err := os.Stat(publicKey); err == nil {
		klog.V(2).Infof("found existing public key at %s", publicKey)
		return
	}

	// no existing keys check for CI variables, create gcloud key files if both exist
	// note only checks if relevant envs are non-empty, no actual key verification checks
	maybePrivateKey, privateKeyEnvSet := os.LookupEnv(ciPrivateKeyEnv)
	if !privateKeyEnvSet {
		klog.V(2).Infof("%s is not set", ciPrivateKeyEnv)
		return
	}
	maybePublicKey, publicKeyEnvSet := os.LookupEnv(ciPublicKeyEnv)
	if !publicKeyEnvSet {
		klog.V(2).Infof("%s is not set", ciPublicKeyEnv)
		return
	}

	if err := fs.CopyFile(maybePrivateKey, t.privateKey); err != nil {
		klog.Warningf("failed to copy %s to %s: %v", maybePrivateKey, t.privateKey, err)
		return
	}

	if err := fs.CopyFile(maybePublicKey, publicKey); err != nil {
		klog.Warningf("failed to copy %s to %s: %v", maybePublicKey, publicKey, err)
	}
}

func (t *Tester) constructArgs() []string {
	if t.RepoRoot == "" {
		// When using pre-built binaries, all parameters get passed through to e2e_node.test
		// with the exact same parameter names. We could have skipped parsing them earlier
		// except that we had to parse first to determine in which mode we are operating.
		//
		// Parsing also provides some early checking for unsupported parameters.
		// The downside is a closer coupling of the node tester to the k/k repo.
		args := []string{
			"--skip-regex=" + t.SkipRegex,
			"--focus-regex=" + t.FocusRegex,
			"--label-filter=" + t.LabelFilter,
			"--gcp-project=" + t.GCPProject,
			"--gcp-zone=" + t.GCPZone,
			"--test-args=" + t.TestArgs,
			"--node-env=" + t.NodeEnv,
			"--delete-instances=" + strconv.FormatBool(t.DeleteInstances),
			"--parallelism=" + strconv.Itoa(t.Parallelism),
			"--image-config-file=" + t.ImageConfigFile,
			"--image-config-dir=" + t.ImageConfigDir,
			"--image-project=" + t.ImageProject,
			"--images=" + t.Images,
			"--instance-metadata=" + t.InstanceMetadata,
			"--user-data-file=" + t.UserDataFile,
			"--instance-type=" + t.InstanceType,
			"--ssh-user=" + t.sshUser,
			"--ssh-key=" + t.privateKey,
			"--timeout=" + t.Timeout.String(),

			// Not relevant: t.UseDockerizedBuild, t.TargetBuildArch
			// Replaced by:
			"--ginkgo-binary=" + t.paths[ginkgo],
			"--kubelet-binary=" + t.paths[kubelet],
			"--e2e-node-binary=" + t.paths[e2eNodeTest],
		}

		if t.RuntimeConfig != "" {
			args = append(args, "--runtime-config="+t.RuntimeConfig)
		}
		return args
	}

	defaultArgs := []string{
		"REMOTE=true",
	}

	argsFromFlags := []string{
		"SKIP=" + t.SkipRegex,
		"FOCUS=" + t.FocusRegex,
		// https://github.com/kubernetes/kubernetes/blob/96be00df69390ed41b8ec22facc43bcbb9c88aae/hack/make-rules/test-e2e-node.sh#L120
		// TODO: this should be configurable without overriding at the gcloud env level
		"CLOUDSDK_CORE_PROJECT=" + t.GCPProject,
		// https://github.com/kubernetes/kubernetes/blob/96be00df69390ed41b8ec22facc43bcbb9c88aae/hack/make-rules/test-e2e-node.sh#L113
		"ZONE=" + t.GCPZone,
		"TEST_ARGS=" + t.TestArgs,
		"NODE_ENV= " + t.NodeEnv,
		"DELETE_INSTANCES=" + strconv.FormatBool(t.DeleteInstances),
		"PARALLELISM=" + strconv.Itoa(t.Parallelism),
		"IMAGE_CONFIG_FILE=" + t.ImageConfigFile,
		"IMAGE_CONFIG_DIR=" + t.ImageConfigDir,
		"IMAGE_PROJECT=" + t.ImageProject,
		"IMAGES=" + t.Images,
		"INSTANCE_METADATA=" + t.InstanceMetadata,
		"USER_DATA_FILE=" + t.UserDataFile,
		"INSTANCE_TYPE=" + t.InstanceType,
		"SSH_USER=" + t.sshUser,
		"SSH_KEY=" + t.privateKey,
		"USE_DOCKERIZED_BUILD=" + strconv.FormatBool(t.UseDockerizedBuild),
		"TARGET_BUILD_ARCH=" + t.TargetBuildArch,
		"TIMEOUT=" + t.Timeout.String(),
		"LABEL_FILTER=" + t.LabelFilter,
	}
	if t.RuntimeConfig != "" {
		argsFromFlags = append(argsFromFlags, "RUNTIME_CONFIG="+t.RuntimeConfig)
	}
	return append(defaultArgs, argsFromFlags...)
}

func (t *Tester) pretestSetup() error {
	if t.UseBuiltBinaries {
		return t.validateLocalBinaries()
	}
	if t.UseBinariesFromPath {
		return t.validateBinariesFromPath()
	}
	if t.UseBinariesFromPackage {
		if err := t.acquirePackages(); err != nil {
			return fmt.Errorf("failed to get packages from published releases: %s", err)
		}
	}
	return nil
}

func (t *Tester) validateLocalBinaries() error {
	klog.V(2).Infof("checking existing test binaries ...")
	for _, binary := range testBinaries {
		path := filepath.Join(t.runDir, string(binary))
		if _, err := os.Stat(path); err != nil {
			logPath := path
			if abspath, err := filepath.Abs(path); err != nil {
				klog.Warningf("failed to convert path %q to absolute path: %v", path, err)
			} else {
				logPath = abspath
			}
			return fmt.Errorf("failed to validate pre-built binary %s (checked at %q): %w", binary, logPath, err)
		}
		klog.V(2).Infof("found existing %s at %s", binary, path)
		t.paths[binary] = path
	}
	return nil
}

func (t *Tester) validateBinariesFromPath() error {
	klog.V(2).Infof("checking for test binaries on PATH...")
	for _, binary := range testBinaries {
		path, err := stdexec.LookPath(string(binary))
		if err != nil {
			return fmt.Errorf("failed to validate binary %s from PATH: %w", binary, err)
		}
		klog.V(2).Infof("found existing %s at %s", binary, path)
		t.paths[binary] = path
	}
	return nil
}

func (t *Tester) Test() error {
	var args []string
	command := ""
	if t.RepoRoot == "" {
		// When using pre-built binaries, `e2e_node.test remote <flags>` itself is the entry point.
		command = t.paths[e2eNodeTest]
		args = []string{"remote"}
	} else {
		// When using the repository, `make test-e2e-node` bootstraps and runs the test.
		command = "make"
		args = []string{"test-e2e-node"}
	}
	args = append(args, t.constructArgs()...)
	cmd := exec.Command(command, args...)
	if t.RepoRoot != "" {
		cmd.SetDir(t.RepoRoot)
	}
	klog.Infof("Running command %q with arguments %q", command, args)
	exec.InheritOutput(cmd)
	return cmd.Run()
}

func Main() {
	t := NewDefaultTester()
	if err := t.Execute(); err != nil {
		klog.Fatalf("failed to run ginkgo tester: %v", err)
	}
}
