#!/usr/bin/env bash
# Copyright 2021 The Kubernetes Authors.
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

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

# this script is invoked in GCB as the entrypoint
# avoid prow potentially not being in the right working directory
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." &> /dev/null && pwd -P)"
cd "${REPO_ROOT}" &> /dev/null
source hack/build/setup-go.sh

make clean

OS_ARCHES=(
  linux_amd64
  darwin_amd64
)

build_os_arch() {
  os="$(echo "$1" | cut -d '_' -f 1)"
  arch="$(echo "$1" | cut -d '_' -f 2)"
  make install-all GOOS="${os}" GOARCH="${arch}" OUT_DIR="${REPO_ROOT}/bin/${os}/${arch}"
  tar -czvf "${REPO_ROOT}/bin/${os}-${arch}.tgz" "${REPO_ROOT}/bin/${os}/${arch}"
}

export -f build_os_arch

# NOTE: disable SC2016 because we _intend_ for these to evaluate later
# shellcheck disable=SC2016
printf '%s\0' "${OS_ARCHES[@]}" | xargs -0 -n 1 -P "${PARALLELISM:-0}" bash -c 'build_os_arch $0'
