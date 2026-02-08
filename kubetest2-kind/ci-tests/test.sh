#!/bin/bash

# Copyright 2026 The Kubernetes Authors.
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

echo "Installing metrics-server via helm"
# helm is temporarily missing, will be added in the future
wget -q "https://get.helm.sh/helm-v4.1.0-linux-$(go env GOARCH).tar.gz" -O helm.tar.gz && \
    tar -xzf helm.tar.gz -C /usr/local/bin --strip-components=1 && \
    rm helm.tar.gz
helm repo add metrics-server https://kubernetes-sigs.github.io/metrics-server/
helm upgrade --install metrics-server metrics-server/metrics-server \
    -n kube-system --set "args={--kubelet-insecure-tls}"


