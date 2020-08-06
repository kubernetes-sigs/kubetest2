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

# Run inside go_container.sh

# cd to the repo root
REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "${REPO_ROOT}"

# Cheat to generate a fixed default value for --artifacts flag.
export ARTIFACTS="${REPO_ROOT}/_artifacts"

function cleanup {
  exit_code=$?
  if [[ "${exit_code}" -ne 0 ]]; then
    echo "Docs are not up-to-date. Please run: make fix"
  fi
  rm -rf "${tmp}"
  exit "${exit_code}"
}

tmp="$(mktemp -d)"
trap 'cleanup' EXIT
cp -r . "${tmp}"
cd "${tmp}"
echo "${tmp}"
git add .
./hack/update/helpdoc_gen.sh
for dir in kubetest2-*/; do
  readmeFile="./${dir}/README.md"
  git diff --quiet -- "${readmeFile}"
done
