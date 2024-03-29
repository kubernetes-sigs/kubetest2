/*
Copyright 2019 The Kubernetes Authors.

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
	"context"
	"io"
	osexec "os/exec"
	"strings"

	"k8s.io/klog/v2"
)

// LocalCmd wraps os/exec.Cmd, implementing the exec.Cmd interface
type LocalCmd struct {
	*osexec.Cmd
}

var _ Cmd = &LocalCmd{}

// LocalCmder is a factory for LocalCmd, implementing Cmder
type LocalCmder struct{}

var _ Cmder = &LocalCmder{}

// Command returns a new exec.Cmd backed by Cmd
func (c *LocalCmder) Command(name string, arg ...string) Cmd {
	klog.V(2).Infof("⚙️ %s %s", name, strings.Join(arg, " "))
	return &LocalCmd{
		Cmd: osexec.Command(name, arg...),
	}
}

// CommandContext returns a new exec.Cmd with the context, backed by Cmd
func (c *LocalCmder) CommandContext(ctx context.Context, name string, arg ...string) Cmd {
	klog.V(2).Infof("⚙️ %s %s", name, strings.Join(arg, " "))
	return &LocalCmd{
		Cmd: osexec.CommandContext(ctx, name, arg...),
	}
}

// SetEnv sets env
func (cmd *LocalCmd) SetEnv(env ...string) Cmd {
	cmd.Env = env
	return cmd
}

// SetStdin sets stdin
func (cmd *LocalCmd) SetStdin(r io.Reader) Cmd {
	cmd.Stdin = r
	return cmd
}

// SetStdout set stdout
func (cmd *LocalCmd) SetStdout(w io.Writer) Cmd {
	cmd.Stdout = w
	return cmd
}

// SetStderr sets stderr
func (cmd *LocalCmd) SetStderr(w io.Writer) Cmd {
	cmd.Stderr = w
	return cmd
}

func (cmd *LocalCmd) SetDir(dir string) Cmd {
	cmd.Dir = dir
	return cmd
}

// Run runs
func (cmd *LocalCmd) Run() error {
	return cmd.Cmd.Run()
}
