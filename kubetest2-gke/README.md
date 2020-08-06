# Command help

This file is auto-generated, please don't update it manually.

```
Usage:
  kubetest2 gke [Flags] [DeployerFlags] -- [TesterArgs]

Flags:
      --artifacts string   directory to put artifacts, defaulting to "${ARTIFACTS:-./_artifacts}". If using the ginkgo tester, this must be an absolute path. (default "/Users/chizhg/go/src/sigs.k8s.io/kubetest2/_artifacts")
      --build              build kubernetes
      --down               tear down the test cluster
  -h, --help               display help
      --test string        test type to run, if unset no tests will run
      --up                 provision the test cluster

DeployerFlags(gke):
      --add_dir_header                       If true, adds the file directory to the header
      --alsologtostderr                      log to standard error as well as files
      --boskos-acquire-timeout-seconds int   How long (in seconds) to hang on a request to Boskos to acquire a resource before erroring (default 300)
      --boskos-location string               If set, manually specifies the location of the boskos server (default "http://boskos.test-pods.svc.cluster.local.")
      --cluster-name strings                 Cluster names separated by comma. Must be set. For multi-project profile, it should be in the format of clusterA:0,clusterB:1,clusterC:2, where the index means the index of the project.
      --create-command string                gcloud subcommand used to create a cluster. Modify if you need to pass arbitrary arguments to create. (default "container clusters create --quiet")
      --environment string                   Container API endpoint to use, one of 'test', 'staging', 'prod', or a custom https:// URL. Defaults to prod if not provided (default "prod")
      --gcp-service-account string           Service account to activate before using gcloud
      --log_backtrace_at traceLocation       when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                       If non-empty, write log files in this directory
      --log_file string                      If non-empty, use this log file
      --log_file_max_size uint               Defines the maximum size a log file can grow to. Unit is megabytes. If the value is 0, the maximum file size is unlimited. (default 1800)
      --logtostderr                          log to standard error instead of files (default true)
      --machine-type string                  For use with gcloud commands to specify the machine type for the cluster. (default "n1-standard-2")
      --network string                       Cluster network. Defaults to the default network if not provided. For multi-project use cases, this will be the Shared VPC network name. (default "default")
      --num-nodes int                        For use with gcloud commands to specify the number of nodes for the cluster. (default 3)
      --project strings                      Project to deploy to separated by comma.
      --projects-requested int               Number of projects to request from boskos. It is only respected if projects is empty, and must be larger than zero  (default 1)
      --region string                        For use with gcloud commands to specify the cluster region.
      --skip_headers                         If true, avoid header prefixes in the log messages
      --skip_log_headers                     If true, avoid headers when opening log files
      --stage string                         Upload binaries to gs://bucket/ci/job-suffix if set
      --stderrthreshold severity             logs at or above this threshold go to stderr (default 2)
      --subnetwork-ranges strings            Subnetwork ranges as required for shared VPC setup as described in https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-shared-vpc#creating_a_network_and_two_subnets.For multi-project profile, it is required and should be in the format of 10.0.4.0/22 10.0.32.0/20 10.4.0.0/14,172.16.4.0/22 172.16.16.0/20 172.16.4.0/22, where the subnetworks configuration for different projectare separated by comma, and the ranges of each subnetwork configuration is separated by space.
  -v, --v Level                              number for the log level verbosity
      --vmodule moduleSpec                   comma-separated list of pattern=N settings for file-filtered logging
      --zone string                          For use with gcloud commands to specify the cluster zone.
```
