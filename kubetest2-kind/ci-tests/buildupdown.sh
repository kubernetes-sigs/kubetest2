#!/bin/bash

# Copyright 2018 The Kubernetes Authors.
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

make install
make install-deployer-kind
make install-tester-ginkgo

# Latest kind could have breaking changes, but as we don't lock
# down Kubernetes or kubeadm either, we have to use the latest,
# potentially not even released revision.
#
# This matches how kind is used elsewhere to test Kubernetes master:
# - https://github.com/kubernetes/test-infra/blob/e1c19ef211ccf4bb242a59e08efc658b511641e6/config/jobs/kubernetes/sig-testing/kubernetes-kind-presubmits.yaml#L26
# - https://github.com/kubernetes/test-infra/blob/e1c19ef211ccf4bb242a59e08efc658b511641e6/config/jobs/kubernetes-sigs/kind/kind.yaml#L58
curl -sSL https://kind.sigs.k8s.io/dl/latest/linux-amd64.tgz | tar xvfz - -C "${GOPATH}/bin"

REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${GOPATH}/src/k8s.io/kubernetes"

# kubetest2 against k/k
kubetest2 kind \
    -v=2 \
    --build \
    --up \
    --down \
    --pre-test-cmd="${REPO_ROOT}/kubetest2-kind/ci-tests/test.sh" \
    --test=ginkgo \
    -- \
    --focus-regex='Secrets should be consumable via the environment' \
    --skip-regex='\[Driver:.gcepd\]|\[Slow\]|\[Serial\]|\[Disruptive\]|\[Flaky\]|\[Feature:.+\]' \
    --use-built-binaries=true \
    --timeout=30m
