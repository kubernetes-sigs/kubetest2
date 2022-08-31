# e2e-framework Tester

This package implements a kubetest2 Tester capable of executing tests written using the Kubernetes-SIGs [e2e-framework](https://github.com/kubernetes-sigs/e2e-framework)

For example, the following would execute tests written using the e2e-framework running on a local KinD cluster:

```
kubetest2 kind -v 2 \
  --up \
  --down \
  --test=e2e-framework \
  --assess='boombap'
  --skip-assessment='network'
```