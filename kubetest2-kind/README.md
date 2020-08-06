# Command help

This file is auto-generated, please don't update it manually.

```
Usage:
  kubetest2 kind [Flags] [DeployerFlags] -- [TesterArgs]

Flags:
      --artifacts string   directory to put artifacts, defaulting to "${ARTIFACTS:-./_artifacts}". If using the ginkgo tester, this must be an absolute path. (default "/Users/chizhg/go/src/sigs.k8s.io/kubetest2/_artifacts")
      --build              build kubernetes
      --down               tear down the test cluster
  -h, --help               display help
      --test string        test type to run, if unset no tests will run
      --up                 provision the test cluster

DeployerFlags(kind):
      --build-type string     --type for kind build node-image
      --cluster-name string   the kind cluster --name (default "kind-kubetest2")
      --config string         --config for kind create cluster
      --image-name string     the image name to use for build and up
      --kubeconfig string     --kubeconfig flag for kind create cluster
      --loglevel string       --loglevel for kind commands
      --verbosity int         --verbosity flag for kind
```
