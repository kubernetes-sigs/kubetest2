# Command help

This file is auto-generated, please don't update it manually.

```
Usage:
  kubetest2 gce [Flags] [DeployerFlags] -- [TesterArgs]

Flags:
      --artifacts string   directory to put artifacts, defaulting to "${ARTIFACTS:-./_artifacts}". If using the ginkgo tester, this must be an absolute path. (default "/Users/chizhg/go/src/sigs.k8s.io/kubetest2/_artifacts")
      --build              build kubernetes
      --down               tear down the test cluster
  -h, --help               display help
      --test string        test type to run, if unset no tests will run
      --up                 provision the test cluster

DeployerFlags(gce):
      --add_dir_header                       If true, adds the file directory to the header
      --alsologtostderr                      log to standard error as well as files
      --boskos-acquire-timeout-seconds int   How long (in seconds) to hang on a request to Boskos to acquire a resource before erroring. (default 300)
      --boskos-location string               If set, manually specifies the location of the boskos server. If unset and boskos is needed, defaults to http://boskos.test-pods.svc.cluster.local. (default "http://boskos.test-pods.svc.cluster.local.")
      --enable-compute-api                   If set, the deployer will enable the compute API for the project during the Up phase. This is necessary if the project has not been used before. WARNING: The currently configured GCP account must have permission to enable this API on the configured project.
      --gcp-project string                   GCP Project to create VMs in. If unset, the deployer will attempt to get a project from boskos.
      --gcp-zone string                      GCP Zone to create VMs in. If unset, kube-up.sh and kube-down.sh defaults apply.
      --legacy-mode                          Set if the provided repo root is the kubernetes/kubernetes repo and not kubernetes/cloud-provider-gcp.
      --log_backtrace_at traceLocation       when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                       If non-empty, write log files in this directory
      --log_file string                      If non-empty, use this log file
      --log_file_max_size uint               Defines the maximum size a log file can grow to. Unit is megabytes. If the value is 0, the maximum file size is unlimited. (default 1800)
      --logtostderr                          log to standard error instead of files (default true)
      --num-nodes int                        The number of nodes in the cluster. (default 3)
      --overwrite-logs-dir                   If set, will overwrite an existing logs directory if one is encountered during dumping of logs. Useful when runnning tests locally.
      --repo-root string                     The path to the root of the local kubernetes/cloud-provider-gcp repo. Necessary to call certain scripts. Defaults to the current directory. If operating in legacy mode, this should be set to the local kubernetes/kubernetes repo.
      --skip_headers                         If true, avoid header prefixes in the log messages
      --skip_log_headers                     If true, avoid headers when opening log files
      --stderrthreshold severity             logs at or above this threshold go to stderr (default 2)
  -v, --v Level                              number for the log level verbosity
      --vmodule moduleSpec                   comma-separated list of pattern=N settings for file-filtered logging
```
