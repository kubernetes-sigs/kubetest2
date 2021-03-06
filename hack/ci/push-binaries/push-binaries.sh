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
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." &> /dev/null && pwd -P)"
# export REPO_ROOT for gcs_upload_version to use in subshells
export REPO_ROOT
cd "${REPO_ROOT}" &> /dev/null

# pass through git details from prow / image builder
if [ -n "${PULL_BASE_SHA:-}" ]; then
  export COMMIT="${PULL_BASE_SHA:?}"
else
  COMMIT="$(git rev-parse HEAD 2>/dev/null)"
  export COMMIT
fi

# short commit is currently 8 characters
SHORT_COMMIT="${COMMIT:0:8}"

# we upload here
BUCKET="${BUCKET:-k8s-staging-kubetest2}"
export BUCKET

# under each of these
VERSIONS=(
  latest
  "${SHORT_COMMIT}"
)

# build the ci binaries
make ci-binaries

gcs_upload_version() {
  echo "uploading CI binaries to gs://${BUCKET}/$1/ ..."
#  gsutil -m cp -P -r "${REPO_ROOT}/bin" "gs://${BUCKET}/$1/"
# only copy the tarballs
  gsutil -m cp -P "${REPO_ROOT}/bin/*.tgz" "gs://${BUCKET}/$1/"
}

export -f gcs_upload_version

# NOTE: disable SC2016 because we _intend_ for these to evaluate later
# shellcheck disable=SC2016
printf '%s\0' "${VERSIONS[@]}" | xargs -0 -n 1 -P "${PARALLELISM:-0}" bash -c 'gcs_upload_version $0'
