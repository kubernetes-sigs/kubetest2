#!/usr/bin/env bash
# Copyright 2022 The Kubernetes Authors.
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

# script to verify go.mod version is <= .go-version
set -o errexit -o nounset -o pipefail

# cd to the repo root
REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "${REPO_ROOT}" &> /dev/null

go_mod_version=$(<go.mod grep ^go | cut -d' ' -f2)
go_version=$(<.go-version cat)
least_version=$(echo -e "${go_mod_version}\n${go_version}" | sort --version-sort | head -n1)
if [[ "${least_version}" != "${go_mod_version}" ]]; then
  echo "go.mod go version '${go_mod_version}' must be less than or equal to .go-version '${go_version}'" >&2
  exit 1
fi
