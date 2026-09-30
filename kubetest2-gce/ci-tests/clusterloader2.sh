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

REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}" &> /dev/null || exit 1

make install
make install-deployer-gce
make install-tester-clusterloader2

# Some CL2 env variables
export PROMETHEUS_SCRAPE_MASTER_KUBELETS=true

cd "${GOPATH}/src/k8s.io/kubernetes"
# kubetest2 against k/k
kubetest2 gce \
    -v=2 \
    --repo-root=. \
    --up \
    --down \
    --test=clusterloader2 \
    --master-size=e2-standard-4 \
    --node-size=e2-standard-8 \
    --num-nodes=1 \
    --env=KUBE_IMAGE_FAMILY=cos-129-lts \
    --env=ETCD_EXTRA_ARGS="--enable-pprof" \
    --env=MAX_PODS_PER_NODE=128 \
    --env=KUBELET_TEST_ARGS="--enable-debugging-handlers --kube-api-qps=100 --kube-api-burst=100" \
    --env=SCHEDULER_TEST_ARGS="--authorization-always-allow-paths=/healthz,/readyz,/livez,/metrics --profiling --contention-profiling --kube-api-qps=100 --kube-api-burst=100" \
    --env=CONTROLLER_MANAGER_TEST_ARGS="--authorization-always-allow-paths=/healthz,/readyz,/livez,/metrics --profiling --contention-profiling --kube-api-qps=100 --kube-api-burst=100" \
    --env=TEST_CLUSTER_DELETE_COLLECTION_WORKERS="--delete-collection-workers=16" \
    -- \
    --provider=gce \
    --repo-root="${GOPATH}"/src/k8s.io/perf-tests \
    --test-configs="${GOPATH}"/src/k8s.io/perf-tests/clusterloader2/testing/node-throughput/config.yaml \
    --test-overrides="${GOPATH}"/src/k8s.io/perf-tests/clusterloader2/testing/overrides/node_docker.yaml
