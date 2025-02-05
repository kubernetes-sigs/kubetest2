# kubetest2

Kubetest2 is a framework for deploying Kubernetes clusters and running end-to-end tests against them.

It is intended to be the next significant iteration of [kubetest]

## Concepts

kubetest2 is effectively split into three independent executables:

- `kubetest2`: discovers and invokes deployers and testers in `PATH`
- `kubetest2-DEPLOYER`: manages the lifecycle of a Kubernetes cluster
- `kubetest2-tester-TESTER`: tests a Kubernetes cluster

The intent behind this design is:
- minimize coupling between deployers and testers
- encourage implementation of new deployers and testers out-of-tree
- keep dependencies / surface area of kubetest2 small

We provide [reference implementations](#reference-implementations) but all
all new implementations should be [external implementations](#external-implementations)

## Installation

To install kubetest2 and all reference deployers and testers:
`go install sigs.k8s.io/kubetest2/...@latest`

To install a specific deployer:
`go install sigs.k8s.io/kubetest2/kubetest2-DEPLOYER@latest` (DEPLOYER can be `gce`, `gke`, etc.)

To install a specific tester:
`go install sigs.k8s.io/kubetest2/kubetest2-tester-TESTER@latest` (TESTER can be `ginkgo`, `exec`, etc.)

## Usage

General usage is of the form:
```
kubetest2 <deployer> [Flags] [DeployerFlags] -- [TesterArgs]
```

**Example**: list all flags for the `noop` deployer and `ginkgo` tester
```
kubetest2 noop --test=ginkgo --help
```

**Example**: deploy a cluster using a local checkout of `kubernetes/kubernetes`, run Conformance tests
```
kubetest2 gce -v 2 \
  --repo-root $KK_REPO_ROOT \
  --gcp-project $YOUR_GCP_PROJECT \
  --legacy-mode \
  --build \
  --up \
  --down \
  --test=ginkgo \
  -- \
  --focus-regex='\[Conformance\]'
```

## Reference Implementations

See individual READMEs for more information

**Deployers**
- [`kubetest2-gce`](/kubetest2-gce)   - use scripts in `kubernetes/cloud-provider-gcp` or `kubernetes/kubernetes`
- [`kubetest2-gke`](/kubetest2-gke)   - use `gcloud containers`
- [`kubetest2-kind`](/kubetest2-kind) - use `kind`
- [`kubetest2-noop`](/kubetest2-noop) - do nothing (to use a pre-existing cluster)

**Testers**
- [`kubetest2-tester-clusterloader2`](/kubetest2-tester-clusterloader2)  - use clusterloader2
- [`kubetest2-tester-exec`](/kubetest2-tester-exec) - exec a given command with the given args / flags
- [`kubetest2-tester-ginkgo`](/kubetest2-tester-ginkgo) - runs e2e tests from `kubernetes/kubernetes`
- [`kubetest2-tester-node`](/kubetest2-tester-node) - runs node e2e tests from `kubernetes/kubernetes`

## External Implementations

**Deployers**
- [`kubetest2-aks`][kubetest2-aks]
- [`kubetest2-kops`][kubetest2-kops]
- [`kubetest2-tf`][kubetest2-tf]
- [`kubetest2-ec2`][kubetest2-ec2]

**Testers**
- [`kubetest2-tester-kops`][kubetest2-tester-kops]

## Support

This project is currently unversioned and unreleased. We make a best-effort attempt to enforce the following:
- `kubetest2` and its reference implementations must work with the in-development version of kubernetes and all [currently supported kubernetes releases][k8s-supported-releases]
  - e.g. no generics until older supported kubernetes version supports generics
  - e.g. ginkgo tester must work with both ginkgo v1 and ginkgo v2
- changes to the following testers must not break jobs in the kubernetes project
  - `kubetest2-tester-exec`
  - `kubetest2-tester-ginkgo`

### Contact

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [Slack](https://kubernetes.slack.com/messages/sig-testing)
- [Mailing List](https://groups.google.com/forum/#!forum/kubernetes-sig-testing)

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).

<!-- links -->
[kubetest]: https://git.k8s.io/test-infra/kubetest
[kubetest2-aks]: https://sigs.k8s.io/cloud-provider-azure/kubetest2-aks
[kubetest2-kops]: https://git.k8s.io/kops/tests/e2e/kubetest2-kops
[kubetest2-tf]: https://github.com/ppc64le-cloud/kubetest2-plugins/tree/master/kubetest2-tf
[kubetest2-ec2]: https://github.com/kubernetes-sigs/provider-aws-test-infra/tree/main/kubetest2-ec2
[kubetest2-tester-kops]: https://git.k8s.io/kops/tests/e2e/kubetest2-tester-kops
[k8s-supported-releases]: https://kubernetes.io/releases/patch-releases/#support-period
