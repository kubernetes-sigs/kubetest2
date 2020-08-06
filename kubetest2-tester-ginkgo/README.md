# Command help

This file is auto-generated, please don't update it manually.

```
      --flake-attempts int            Make up to this many attempts to run each spec. (default 1)
      --focus-regex string            Regular expression of jobs to focus on.
  -h, --help                          
      --parallel int                  Run this many tests in parallel at once. (default 1)
      --skip-regex string             Regular expression of jobs to skip.
      --test-package-bucket string    The bucket which release tars will be downloaded from to acquire the test package. Defaults to the main kubernetes project bucket. (default "kubernetes-release")
      --test-package-version string   The ginkgo tester uses a test package made during the kubernetes build. The tester downloads this test package from one of the release tars published to GCS. Defaults to latest. Use "gsutil ls gs://kubernetes-release/release/" to find release names. Example: v1.20.0-alpha.0
```
