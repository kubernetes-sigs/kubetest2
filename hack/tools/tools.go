// +build tools

/*
Package tools is used to track binary dependencies with go modules
https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
*/
package tools

import (
	// linter(s)
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"

	// test runner
	_ "gotest.tools/gotestsum"
)
