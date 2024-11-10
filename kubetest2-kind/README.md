# Kubetest2 KIND Deployer

This component of kubetest2 is responsible for test cluster lifecycles for clusters deployed using kind.

## Usage

Currently, the kind deployer is capable of building and deploying a cluster using kind.

```
kubetest2 kind --build --up --down --test=exec -- kubectl get all -A
```

See the usage (`--help`) for more options.

## Implementation
The deployer builds a kind node image and is essentially a Golang wrapper for building e2e dependencies as in `e2e-k8s.sh` located [here](https://github.com/kubernetes-sigs/kind/blob/main/hack/ci/e2e-k8s.sh#L72-L86) in  [kind](https://github.com/kubernetes-sigs/kind/tree/main)
