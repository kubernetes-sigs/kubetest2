# kubetest2

Kubetest2 is the framework for launching and running end-to-end tests on Kubernetes.
It is intended to be the next significant iteration of [kubetest](https://github.com/kubernetes/test-infra/tree/master/kubetest).

## Installation
To install core and all deployers and testers:
`GO111MODULE=on go get sigs.k8s.io/kubetest2/...@latest`

To install a specific deployer:
`GO111MODULE=on go get sigs.k8s.io/kubetest2/kubetest2-DEPLOYER@latest` (DEPLOYER can be `gce`, `gke`, etc.)

To install a sepcific tester:
`GO111MODULE=on go get sigs.k8s.io/kubetest2/kubetest2-tester-TESTER@latest` (TESTER can be `ginkgo`, `exec`, etc.)

## Usage
An example run of the Ginkgo conformance suite against your local version of the k/k repo deployed to GCE looks as follows:
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

See READMEs specific to each deployer and tester for information about each. Usage (`--help`) should also be referenced.

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [Slack](https://kubernetes.slack.com/messages/sig-testing)
- [Mailing List](https://groups.google.com/forum/#!forum/kubernetes-sig-testing)

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).
