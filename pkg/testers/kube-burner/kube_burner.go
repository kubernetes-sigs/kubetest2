/*
Copyright 2026 The Kubernetes Authors.

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

package kubeburner

import (
	"flag"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"

	"github.com/kballard/go-shellquote"
	"github.com/octago/sflags/gen/gpflag"
	"k8s.io/klog/v2"

	"sigs.k8s.io/kubetest2/pkg/exec"
	"sigs.k8s.io/kubetest2/pkg/testers"
)

var GitTag string

type Tester struct {
	Workload       string `desc:"kube-burner workload to run (e.g. node-density, cluster-density-v2)."`
	WorkDir        string `desc:"Base directory containing kube-burner workload configs. The config file is resolved as <workdir>/<workload>/config.yaml."`
	KubeBurnerPath string `desc:"Path to the kube-burner binary. If not specified, kube-burner is looked up in PATH."`
	KubeConfig     string `desc:"Path to kubeconfig. If specified will override the path exposed by the kubetest2 deployer."`
	LogLevel       string `desc:"kube-burner log level (debug, info, warn, error, fatal)."`
	UUID           string `desc:"Custom UUID for the benchmark run."`
	SkipTLSVerify  bool   `desc:"Skip TLS verification for kube-apiserver connections."`
	ExtraArgs      string `flag:"~extra-args" desc:"Additional arguments to pass to kube-burner (https://github.com/kube-burner/kube-burner)."`
}

func NewDefaultTester() *Tester {
	return &Tester{
		LogLevel: "debug",
	}
}

func (t *Tester) resolveKubeconfig() (string, error) {
	// 1. User explicitly specified --kubeconfig flag
	if t.KubeConfig != "" {
		klog.V(2).Infof("using kubeconfig from --kubeconfig flag: %s", t.KubeConfig)
		return t.resolveAbsPath(t.KubeConfig)
	}

	// 2. KUBECONFIG env var (set by the kubetest2 framework from the deployer)
	if kconfig := os.Getenv("KUBECONFIG"); kconfig != "" {
		klog.V(2).Infof("using kubeconfig from KUBECONFIG env: %s", kconfig)
		return t.resolveAbsPath(kconfig)
	}

	// 3. Fall back to ~/.kube/config
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory for default kubeconfig: %v", err)
	}
	defaultPath := filepath.Join(home, ".kube", "config")
	if _, err := os.Stat(defaultPath); err != nil {
		return "", fmt.Errorf("no kubeconfig found: set --kubeconfig, KUBECONFIG env, or ensure ~/.kube/config exists: %v", err)
	}
	klog.V(2).Infof("using default kubeconfig: %s", defaultPath)
	return defaultPath, nil
}

func (t *Tester) resolveAbsPath(path string) (string, error) {
	if !filepath.IsAbs(path) {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("failed to resolve absolute path for kubeconfig %q: %v", path, err)
		}
		path = abs
	}
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("kubeconfig not found at %q: %v", path, err)
	}
	return path, nil
}

func (t *Tester) Test() error {
	if t.Workload == "" {
		return fmt.Errorf("--workload is required: kube-burner workload to run")
	}

	kubeBurnerPath := t.KubeBurnerPath
	if kubeBurnerPath == "" {
		p, err := osexec.LookPath("kube-burner")
		if err != nil {
			return fmt.Errorf("kube-burner not found in PATH, use --kube-burner-path to specify the binary location: %v", err)
		}
		kubeBurnerPath = p
	}

	kubeconfig, err := t.resolveKubeconfig()
	if err != nil {
		return err
	}

	workloadDir := filepath.Join(t.WorkDir, t.Workload)
	configPath := filepath.Join(workloadDir, "config.yaml")

	args := []string{
		"init",
		"--config=" + configPath,
		"--kubeconfig=" + kubeconfig,
	}

	overridesPath := filepath.Join(workloadDir, "overrides.yaml")
	if _, err := os.Stat(overridesPath); err == nil {
		args = append(args, "--user-data="+overridesPath)
		klog.V(2).Infof("found overrides file: %s", overridesPath)
	}
	if t.LogLevel != "" {
		args = append(args, "--log-level="+t.LogLevel)
	}
	if t.UUID != "" {
		args = append(args, "--uuid="+t.UUID)
	}
	if t.SkipTLSVerify {
		args = append(args, "--skip-tls-verify")
	}

	parsedExtraArgs, err := shellquote.Split(t.ExtraArgs)
	if err != nil {
		return fmt.Errorf("error parsing --extra-args: %v", err)
	}
	args = append(args, parsedExtraArgs...)

	cmd := exec.Command(kubeBurnerPath, args...)
	exec.InheritOutput(cmd)
	klog.V(2).Infof("running kube-burner %s", args)
	return cmd.Run()
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
	if err := testers.WriteVersionToMetadata(GitTag, ""); err != nil {
		return err
	}
	return t.Test()
}

func Main() {
	t := NewDefaultTester()
	if err := t.Execute(); err != nil {
		klog.Fatalf("failed to run kube-burner tester: %v", err)
	}
}
