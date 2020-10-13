#!/usr/bin/env bash
# Copyright 2019 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit -o nounset -o pipefail

# cd to the repo root and setup go
REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "${REPO_ROOT}" &> /dev/null
source hack/build/setup-go.sh

function cleanup {
  exit_code=$?
  if [[ "${exit_code}" -ne 0 ]]; then
    echo "Modules are not tidied. Please run: make fix"
  fi
  rm -rf "${tmp}"
  exit "${exit_code}"
}

tmp="$(mktemp -d)"
trap 'cleanup' EXIT
cp -r . "${tmp}"
cd "${tmp}" &> /dev/null
git add .
go mod tidy
git diff --quiet -- go.mod go.sum
