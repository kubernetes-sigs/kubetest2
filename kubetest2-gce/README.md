# Kubetest2 GCE Deployer

This component of kubetest2 is responsible for test cluster lifecycles for clusters deployed to GCE VMs.

The [original design proposal](https://docs.google.com/document/d/157nSQNyy9cOjw4izG0rUs_9z9Suy31JQtng_g2peTsw/edit#heading=h.5irk4csrpu0y) has a great deal of detail about the motivations for the deployer and detailed evaluations of the original (kubetest) deployer it replaces.

## Usage

Currently, the GCE deployer must be running on a system with a version of k/k or the cloud-provider-gcp repository cloned. A simple run without running tests looks as follows:

```
kubetest2 gce --gcp-project $TARGETPROJECT --repo-root $CLONEDREPOPATH --build --up --down
```

If targeting k/k instead of cloud-provider-gcp, you must add `--legacy-mode` so the deployer knows how to build the code.

The deployer supports Boskos, so `--gcp-project` can be skipped if there is an available Boskos instance running.

See the usage (`--help`) for more options.
