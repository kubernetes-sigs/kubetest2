/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package deployer

import (
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sync/errgroup"
	"k8s.io/klog/v2"

	"sigs.k8s.io/kubetest2/pkg/exec"
	"sigs.k8s.io/kubetest2/pkg/metadata"
)

// Deployer implementation methods below
func (d *Deployer) Up() error {
	if err := d.Init(); err != nil {
		return err
	}

	if err := d.CreateNetwork(); err != nil {
		return err
	}
	if err := d.CreateClusters(); err != nil {
		if d.RepoRoot == "" {
			klog.Warningf("repo-root not supplied, skip dumping cluster logs")
		}
		if err := d.DumpClusterLogs(); err != nil {
			klog.Warningf("Dumping cluster logs at the end of Up() failed: %v", err)
		}
		return fmt.Errorf("error creating the clusters: %w", err)
	}

	if err := d.TestSetup(); err != nil {
		if d.RepoRoot == "" {
			klog.Warningf("repo-root not supplied, skip dumping cluster logs")
		}
		if err := d.DumpClusterLogs(); err != nil {
			klog.Warningf("Dumping cluster logs at the end of Up() failed: %v", err)
		}
		return fmt.Errorf("error running setup for the tests: %w", err)
	}

	return nil
}

func (d *Deployer) CreateClusters() error {
	klog.V(2).Infof("Environment: %v", os.Environ())

	totalTryCount := math.Max(float64(len(d.Regions)), float64(len(d.Zones)))
	for retryCount := 0; retryCount < int(totalTryCount); retryCount++ {
		d.retryCount = retryCount
		shouldRetry, err := d.tryCreateClusters(retryCount)
		if !shouldRetry {
			return err
		}
	}

	return nil
}

func (d *Deployer) tryCreateClusters(retryCount int) (shouldRetry bool, err error) {
	shouldRetry = false
	if err = d.CreateSubnets(); err != nil {
		return
	}
	if err = d.SetupNetwork(); err != nil {
		return
	}

	eg := new(errgroup.Group)
	locationArg := locationFlag(d.Regions, d.Zones, retryCount)
	for i := range d.Projects {
		project := d.Projects[i]
		clusters := d.projectClustersLayout[project]
		subNetworkArgs := subNetworkArgs(d.Autopilot, d.Projects, regionFromLocation(d.Regions, d.Zones, retryCount), d.Network, i)
		for j := range clusters {
			cluster := clusters[j]
			eg.Go(
				func() error {
					return d.CreateCluster(project, cluster, subNetworkArgs, locationArg)
				},
			)
		}
	}

	if err = eg.Wait(); err != nil {
		// If the error is retryable and it is not the last region/zone that
		// can be retried, perform cleanups in the background and retry
		// cluster creation in the next available region/zone.
		if d.isRetryableError(err) && retryCount != d.totalTryCount-1 {
			shouldRetry = true
			go func() {
				d.DeleteClusters(retryCount)
				if err := d.DeleteSubnets(retryCount); err != nil {
					log.Printf("Warning: error encountered deleting subnets: %v", err)
				}
			}()
		} else {
			err = fmt.Errorf("error creating clusters: %v", err)
		}
	}

	if err == nil {
		err = d.clustersSetenv(location(d.Regions, d.Zones, retryCount))
	}

	return
}

// clustersSetenv sets environment variables identifying the created clusters.
func (d *Deployer) clustersSetenv(location string) error {
	projects, names, locations := []string{}, []string{}, []string{}
	for i := range d.Projects {
		project := d.Projects[i]
		clusters := d.projectClustersLayout[project]
		for j := range clusters {
			cluster := clusters[j]
			projects = append(projects, project)
			names = append(names, cluster.name)
			locations = append(locations, location)
		}
	}
	if err := os.Setenv("GKE_CLUSTER_PROJECTS", strings.Join(projects, ",")); err != nil {
		return fmt.Errorf("error setting GKE_CLUSTER_PROJECTS: %v", err)
	}
	if err := os.Setenv("GKE_CLUSTER_NAMES", strings.Join(names, ",")); err != nil {
		return fmt.Errorf("error setting GKE_CLUSTER_NAMES: %v", err)
	}
	if err := os.Setenv("GKE_CLUSTER_LOCATIONS", strings.Join(locations, ",")); err != nil {
		return fmt.Errorf("error setting GKE_CLUSTER_LOCATIONS: %v", err)
	}
	return nil
}

// isRetryableError checks if the error happens during cluster creation can be potentially solved by retrying or not.
func (d *Deployer) isRetryableError(err error) bool {
	for _, regx := range d.retryableErrorPatternsCompiled {
		if regx.MatchString(err.Error()) {
			return true
		}
	}
	return false
}

func (d *Deployer) CreateCluster(project string, cluster cluster, subNetworkArgs []string, locationArg string) error {
	privateClusterArgs := []string{}
	if d.PrivateClusterAccessLevel != "" {
		privateClusterArgs = getPrivateClusterArgs(d.Projects, d.Network, d.PrivateClusterAccessLevel, d.privateClusterMasterIPRangesInternal[d.retryCount], cluster, d.Autopilot)
	}
	// Create the cluster
	args := d.createCommand()
	args = append(args,
		"--project="+project,
		locationArg,
		"--network="+transformNetworkName(d.Projects, d.Network),
	)
	// A few args are not supported in GKE Autopilot cluster creation, so they should be left unset.
	// https://cloud.google.com/sdk/gcloud/reference/container/clusters/create-auto
	if !d.Autopilot {
		if d.MachineType != "" {
			args = append(args, "--machine-type="+d.MachineType)
		}
		args = append(args, "--num-nodes="+strconv.Itoa(d.NumNodes))
		if d.ImageType != "" {
			args = append(args, "--image-type="+d.ImageType)
		}
		if d.WorkloadIdentityEnabled {
			args = append(args, fmt.Sprintf("--workload-pool=%s.svc.id.goog", project))
		}
	}

	if d.ReleaseChannel != "" {
		args = append(args, "--release-channel="+d.ReleaseChannel)
		if d.ClusterVersion == "latest" {
			// If latest is specified, get the latest version from server config for this channel.
			actualVersion, err := resolveLatestVersionInChannel(locationArg, d.ReleaseChannel)
			if err != nil {
				return err
			}
			klog.V(0).Infof("Using the latest version %q in %q channel", actualVersion, d.ReleaseChannel)
			args = append(args, "--cluster-version="+actualVersion)
		} else {
			args = append(args, "--cluster-version="+d.ClusterVersion)
		}
	} else {
		args = append(args, "--cluster-version="+d.ClusterVersion)
		releaseChannel, err := resolveReleaseChannelForClusterVersion(d.ClusterVersion, locationArg)
		if err != nil {
			klog.Warningf("error resolving the release channel for %q: %v, will proceed with no channel", d.ClusterVersion, err)
		} else {
			args = append(args, "--release-channel="+releaseChannel)
		}
	}
	args = append(args, subNetworkArgs...)
	args = append(args, privateClusterArgs...)
	args = append(args, cluster.name)
	output, err := runWithOutputAndReturn(exec.Command("gcloud", args...))
	if err != nil {
		//parse output for match with regex error
		return fmt.Errorf("error creating cluster: %v, output: %q", err, output)
	}

	if d.WindowsEnabled {
		args := d.createNodePoolCommand(project, cluster, locationArg, "windows-pool", d.WindowsImageType, d.WindowsMachineType, d.WindowsNumNodes)
		output, err := runWithOutputAndReturn(exec.Command("gcloud", args...))
		if err != nil {
			return fmt.Errorf("error creating windows node-pool: %v, output: %q", err, output)
		}
	}

	eg := new(errgroup.Group)
	// serialize extra nodepool creates by default.
	eg.SetLimit(1)
	if d.NodePoolCreateConcurrency > 1 {
		eg.SetLimit(d.NodePoolCreateConcurrency)
	}

	for _, enp := range d.extraNodePoolSpecs {
		enp := enp
		eg.Go(func() error {
			args := d.createNodePoolCommand(project, cluster, locationArg, enp.Name, enp.ImageType, enp.MachineType, enp.NumNodes)
			output, err := runWithOutputAndReturn(exec.Command("gcloud", args...))
			if err != nil {
				return fmt.Errorf("error creating nodepool %q: %v, output: %q", enp.Name, err, output)
			}
			return nil
		})
	}

	return eg.Wait()
}

func (d *Deployer) createCommand() []string {
	// Use the --create-command flag if it's explicitly specified.
	if d.CreateCommandFlag != "" {
		return strings.Fields(d.CreateCommandFlag)
	}

	fs := make([]string, 0)
	if d.GcloudCommandGroup != "" {
		fs = append(fs, d.GcloudCommandGroup)
	}

	fs = append(fs, "container", "clusters")
	if d.Autopilot {
		fs = append(fs, "create-auto")
	} else {
		fs = append(fs, "create")
	}
	fs = append(fs, "--quiet")
	fs = append(fs, strings.Fields(d.GcloudExtraFlags)...)
	return fs
}

func (d *Deployer) createNodePoolCommand(project string, cluster cluster, locationArg, nodePoolName, imageType string, machineType string, numNodes int) []string {
	fs := make([]string, 0)
	fs = append(fs, "container", "node-pools", "create", nodePoolName)
	fs = append(fs, "--quiet")
	fs = append(fs, "--cluster="+cluster.name)
	fs = append(fs, "--project="+project)
	fs = append(fs, locationArg)
	if imageType != "" {
		fs = append(fs, "--image-type="+imageType)
	}
	if machineType != "" {
		fs = append(fs, "--machine-type="+machineType)
	}
	fs = append(fs, "--num-nodes="+strconv.Itoa(numNodes))

	return fs
}

func (d *Deployer) IsUp() (up bool, err error) {
	if err := d.PrepareGcpIfNeeded(d.Projects[0]); err != nil {
		return false, err
	}

	for _, project := range d.Projects {
		for _, cluster := range d.projectClustersLayout[project] {
			if err := getClusterCredentials(project, locationFlag(d.Regions, d.Zones, d.retryCount), cluster.name); err != nil {
				return false, err
			}

			// naively assume that if the api server reports nodes, the cluster is up
			lines, err := exec.CombinedOutputLines(
				exec.RawCommand("kubectl get nodes -o=name"),
			)
			if err != nil {
				return false, metadata.NewJUnitError(err, strings.Join(lines, "\n"))
			}
			if len(lines) == 0 {
				return false, fmt.Errorf("project had no nodes active: %s", project)
			}
		}
	}

	return true, nil
}

func (d *Deployer) TestSetup() error {
	if d.testPrepared {
		// Ensure setup is a singleton.
		return nil
	}

	if _, err := d.Kubeconfig(); err != nil {
		return err
	}
	if err := d.GetInstanceGroups(); err != nil {
		return err
	}
	if err := d.EnsureFirewallRules(); err != nil {
		return err
	}
	if err := d.EnsureNat(); err != nil {
		return err
	}
	d.testPrepared = true
	return nil
}

// Kubeconfig returns a path to a kubeconfig file for the cluster in
// a temp directory, creating one if one does not exist.
// It also sets the KUBECONFIG environment variable appropriately.
func (d *Deployer) Kubeconfig() (string, error) {
	if d.kubecfgPath != "" {
		return d.kubecfgPath, nil
	}

	tmpdir, err := os.MkdirTemp("", "kubetest2-gke")
	if err != nil {
		return "", err
	}

	kubecfgFiles := make([]string, 0)
	for _, project := range d.Projects {
		for _, cluster := range d.projectClustersLayout[project] {
			filename := filepath.Join(tmpdir, fmt.Sprintf("kubecfg-%s-%s", project, cluster.name))
			if err := os.Setenv("KUBECONFIG", filename); err != nil {
				return "", err
			}
			if err := getClusterCredentials(project, locationFlag(d.Regions, d.Zones, d.retryCount), cluster.name); err != nil {
				return "", err
			}
			kubecfgFiles = append(kubecfgFiles, filename)
		}
	}

	d.kubecfgPath = strings.Join(kubecfgFiles, string(os.PathListSeparator))
	return d.kubecfgPath, nil
}

// verifyCommonFlags validates flags for up phase.
func (d *Deployer) VerifyUpFlags() error {
	if len(d.Projects) == 0 {
		for _, num := range d.BoskosProjectsRequested {
			d.totalBoskosProjectsRequested += num
		}
		if d.totalBoskosProjectsRequested <= 0 {
			return fmt.Errorf("either --project or --projects-requested with a value larger than 0 must be set for GKE deployment")
		}
		if len(d.BoskosProjectsRequested) != len(d.BoskosResourceType) {
			return fmt.Errorf("the length of --project-requested and --boskos-resource-type must be the same")
		}
	}

	if len(d.Clusters) == 0 {
		if len(d.Projects) > 1 || d.totalBoskosProjectsRequested > 1 {
			return fmt.Errorf("explicit --cluster-name must be set for multi-project profile")
		}
		if err := d.ClusterOptions.Validate(); err != nil {
			return err
		}
		d.Clusters = generateClusterNames(d.ClusterOptions.NumClusters, d.Kubetest2CommonOptions.RunID())
	} else {
		klog.V(0).Infof("explicit --cluster-name specified, ignoring --num-clusters")
	}
	if err := d.VerifyNetworkFlags(); err != nil {
		return err
	}
	if err := d.VerifyLocationFlags(); err != nil {
		return err
	}
	if d.NumNodes <= 0 {
		return fmt.Errorf("--num-nodes must be larger than 0")
	}
	if err := validateVersion(d.ClusterVersion); err != nil {
		return err
	}
	if err := validateReleaseChannel(d.ReleaseChannel); err != nil {
		return err
	}

	for _, np := range d.ExtraNodePool {
		// defaults
		enp := &extraNodepool{}

		if err := buildExtraNodePoolOptions(np, enp); err != nil {
			return fmt.Errorf("invalid extra nodepool spec %q: %v", np, err)
		}
	}

	return nil
}

func generateClusterNames(numClusters int, uid string) []string {
	clusters := make([]string, numClusters)
	for i := 1; i <= numClusters; i++ {
		// Naming convention: https://cloud.google.com/sdk/gcloud/reference/container/clusters/create#POSITIONAL-ARGUMENTS
		// must start with an alphabet, max length 40

		// 4 characters for kt2- prefix (short for kubetest2)
		const fixedClusterNamePrefix = "kt2-"
		// 3 characters -99 suffix
		clusterNameSuffix := strconv.Itoa(i)
		// trim the uid to only use the first 33 characters
		var id string
		if uid != "" {
			const maxIDLength = 33
			if len(uid) > maxIDLength {
				id = uid[:maxIDLength]
			} else {
				id = uid
			}
			id += "-"
		}
		clusters[i-1] = fixedClusterNamePrefix + id + clusterNameSuffix
	}
	return clusters
}
