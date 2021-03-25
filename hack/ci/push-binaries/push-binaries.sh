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

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "${REPO_ROOT}" &> /dev/null


# pass through git details from prow / image builder
if [ -n "${PULL_BASE_SHA:-}" ]; then
  export COMMIT="${PULL_BASE_SHA:?}"
else
  COMMIT="$(git rev-parse --short HEAD 2>/dev/null)"
  export COMMIT
fi

# we upload here
BUCKET="${BUCKET:-k8s-staging-kubetest2}"
export BUCKET

# under each of these
VERSIONS=(
  latest
  "${COMMIT}"
)

# build the ci binaries
make ci-binaries

gcs_upload_version() {
  echo "uploading CI binaries to gs://${BUCKET}/$1/ ..."
  gsutil -m cp -P -r "${REPO_ROOT}/bin" "gs://${BUCKET}/$1/"
}

export -f gcs_upload_version

# NOTE: disable SC2016 because we _intend_ for these to evaluate later
# shellcheck disable=SC2016
printf '%s\0' "${VERSIONS[@]}" | xargs -0 -n 1 -P "${PARALLELISM:-0}" bash -c 'gcs_upload_version $0'
