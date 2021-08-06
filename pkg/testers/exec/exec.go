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

package exec

import (
	"fmt"
	"os"
	"strings"

	"github.com/octago/sflags/gen/gpflag"
	"k8s.io/klog"

	"sigs.k8s.io/kubetest2/pkg/process"
)

var GitTag string

type Tester struct {
	argv []string
}

const usage = `kubetest2 --test=exec --  [TestCommand] [TestArgs]
  TestCommand: the command to invoke for testing
  TestArgs:    arguments passed to test command
`

func (t *Tester) Execute() error {
	fs, err := gpflag.Parse(t)
	if err != nil {
		return fmt.Errorf("failed to initialize tester: %v", err)
	}

	fs.Usage = func() {
		fmt.Print(usage)
	}

	if len(os.Args) < 2 {
		fs.Usage()
		return nil
	}

	// gracefully handle -h or --help if it is the only argument
	help := fs.BoolP("help", "h", false, "")
	// we don't care about errors, only if -h / --help was set
	_ = fs.Parse(os.Args[1:2])

	if *help {
		fs.Usage()
		return nil
	}

	t.argv = os.Args[1:]
	return t.Test()
}

func expandEnv(args []string) []string {
	expandedArgs := make([]string, len(args))
	for i, arg := range args {
		// best effort handle literal dollar for backward compatibility
		// this is not an all-purpose shell special character handler
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
	return process.ExecJUnit(expandedArgs[0], expandedArgs[1:], os.Environ())
}

func NewDefaultTester() *Tester {
	return &Tester{}
}

func Main() {
	t := NewDefaultTester()
	if err := t.Execute(); err != nil {
		klog.Fatalf("failed to run exec tester: %v", err)
	}
}
