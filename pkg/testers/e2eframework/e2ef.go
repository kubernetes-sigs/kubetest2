/*
Copyright 2020 The Kubernetes Authors.

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

package e2eframework

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/octago/sflags/gen/gpflag"
	"k8s.io/klog/v2"

	"sigs.k8s.io/kubetest2/pkg/exec"
	"sigs.k8s.io/kubetest2/pkg/testers"
)

var GitTag string

type Tester struct {
	Packages string `desc:"A space-separated list of package to test"`
	argv     []string
}

const usage = `kubetest2 --test=e2e-framework [e2e-framework args]
  e2e-framework args:    arguments passed to the command running the e2e-framework tests
`

func (t *Tester) Execute() error {
	fs, err := gpflag.Parse(t)
	if err != nil {
		return fmt.Errorf("failed to initialize e2e-framework tester: %v", err)
	}

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

	if err := testers.WriteVersionToMetadata(GitTag); err != nil {
		return err
	}
	return t.Test()
}

// expandEnv copied from exec-tester
// TODO - replace with gexe
func expandEnv(args []string) []string {
	expandedArgs := make([]string, len(args))
	for i, arg := range args {
		if strings.Contains(arg, `\$`) {
			expandedArgs[i] = strings.ReplaceAll(arg, `\$`, `$`)
		} else {
			expandedArgs[i] = os.ExpandEnv(arg)
		}
	}
	return expandedArgs
}

func (t *Tester) Test() error {
	expandedArgs := expandEnv(t.argv)
	cmd := exec.Command("go", append(append([]string{"test"}, expandedArgs...), t.Packages)...)
	exec.InheritOutput(cmd)
	return cmd.Run()
}

func New() *Tester {
	return &Tester{}
}

func Main() {
	t := New()
	if err := t.Execute(); err != nil {
		klog.Fatalf("failed to run e2e-framework tester: %v", err)
	}
}
