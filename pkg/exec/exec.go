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
	"bufio"
	"bytes"
	"context"
	"io"
	"os"

	shell "github.com/kballard/go-shellquote"
)

// Cmd abstracts over running a command somewhere, this is useful for testing
type Cmd interface {
	Run() error
	// Each entry should be of the form "key=value"
	SetEnv(...string) Cmd
	SetStdin(io.Reader) Cmd
	SetStdout(io.Writer) Cmd
	SetStderr(io.Writer) Cmd
	SetDir(string) Cmd
}

// Cmder abstracts over creating commands
type Cmder interface {
	// command, args..., just like os/exec.Cmd
	Command(string, ...string) Cmd
	CommandContext(context.Context, string, ...string) Cmd
}

// DefaultCmder is a LocalCmder instance used for convenience, packages
// originally using os/exec.Command can instead use pkg/kind/exec.Command
// which forwards to this instance
// TODO(bentheelder): swap this for testing
// TODO(bentheelder): consider not using a global for this :^)
var DefaultCmder = &LocalCmder{}

// Command is a convenience wrapper over DefaultCmder.Command
func Command(command string, args ...string) Cmd {
	return DefaultCmder.Command(command, args...)
}

func CommandContext(ctx context.Context, command string, args ...string) Cmd {
	return DefaultCmder.CommandContext(ctx, command, args...)
}

func RawCommand(raw string) Cmd {
	cmdSplit, err := shell.Split(raw)
	// If failed to split, just return the raw string as the command.
	if len(cmdSplit) == 0 || err != nil {
		return DefaultCmder.Command(raw)
	}
	return DefaultCmder.Command(cmdSplit[0], cmdSplit[1:]...)
}

func RawCommandContext(ctx context.Context, raw string) Cmd {
	cmdSplit, err := shell.Split(raw)
	// If failed to split, just return the raw string as the command.
	if len(cmdSplit) == 0 || err != nil {
		return DefaultCmder.CommandContext(ctx, raw)
	}
	return DefaultCmder.CommandContext(ctx, cmdSplit[0], cmdSplit[1:]...)
}

// Output is for compatibility with cmd.Output.
func Output(cmd Cmd) ([]byte, error) {
	var buff bytes.Buffer
	cmd.SetStdout(&buff)
	err := cmd.Run()
	return buff.Bytes(), err
}

// OutputLines is like os/exec's cmd.Output(),
// but over our Cmd interface, and instead of returning the byte buffer of
// stdout, it scans it for lines and returns a slice of output lines
func OutputLines(cmd Cmd) (lines []string, err error) {
	var buff bytes.Buffer
	cmd.SetStdout(&buff)
	err = cmd.Run()
	scanner := bufio.NewScanner(&buff)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, err
}

// CombinedOutputLines is like os/exec's cmd.CombinedOutput(),
// but over our Cmd interface, and instead of returning the byte buffer of
// stderr + stdout, it scans these for lines and returns a slice of output lines
func CombinedOutputLines(cmd Cmd) (lines []string, err error) {
	var buff bytes.Buffer
	cmd.SetStdout(&buff)
	cmd.SetStderr(&buff)
	err = cmd.Run()
	scanner := bufio.NewScanner(&buff)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, err
}

// SetOutput sets cmd's output to write to the given Writer.
func SetOutput(cmd Cmd, stdoutWriter, stderrWriter io.Writer) {
	cmd.SetStdout(stdoutWriter)
	cmd.SetStderr(stderrWriter)
}

// InheritOutput sets cmd's output to write to the current process's stdout and stderr
func InheritOutput(cmd Cmd) {
	cmd.SetStderr(os.Stderr)
	cmd.SetStdout(os.Stdout)
}

// NoOutput ignores all output from the command.
func NoOutput(cmd Cmd) {
	cmd.SetStdout(io.Discard)
	cmd.SetStderr(io.Discard)
}
