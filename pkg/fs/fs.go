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

package fs

import (
	"io"
	"os"
)

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) (err error) {
	// get source information
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	return copyFile(src, dst, info)
}

func copyFile(src, dst string, info os.FileInfo) error {
	// open src for reading
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	// create dst file
	// this is like f, err := os.Create(dst); os.Chmod(f.Name(), src.Mode())
	out, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	// make sure we close the file
	defer func() {
		closeErr := out.Close()
		// if we weren't returning an error
		if err == nil {
			err = closeErr
		}
	}()
	// actually copy
	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	err = out.Sync()
	return err
}
