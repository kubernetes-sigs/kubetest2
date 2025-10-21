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
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"k8s.io/klog/v2"

	"sigs.k8s.io/kubetest2/pkg/exec"
)

const networkUserPolicyTemplate = `
bindings:
- members:
  - serviceAccount:%s
  - serviceAccount:%s
  role: roles/compute.networkUser
etag: %s
`

func (d *Deployer) VerifyNetworkFlags() error {

	// Verify private cluster args.
	if d.PrivateClusterAccessLevel != "" {
		if d.PrivateClusterAccessLevel != string(no) &&
			d.PrivateClusterAccessLevel != string(limited) && d.PrivateClusterAccessLevel != string(unrestricted) {
			return fmt.Errorf("--private-cluster-access-level must be one of %v", []string{"", string(no), string(limited), string(unrestricted)})
		}
		if len(d.PrivateClusterMasterIPRanges) != len(d.Clusters)*d.totalTryCount {
			return fmt.Errorf("the number of master ip ranges provided via --private-cluster-master-ip-range "+
				"should be the same as the number of clusters times the total try count : %d!=%d", len(d.PrivateClusterMasterIPRanges), len(d.Clusters)*d.totalTryCount)
		}
		if err := assertNoOverlaps(d.PrivateClusterMasterIPRanges); err != nil {
			return fmt.Errorf("error in private cluster master ip ranges: %v", err)
		}
	}

	numProjects := len(d.Projects)
	if numProjects == 0 {
		numProjects = d.totalBoskosProjectsRequested
	}

	// Verify for multi-project profile.
	if numProjects > 1 {
		if d.Network == "default" {
			return errors.New("the default network cannot be used for multi-project profile")
		}

		if len(d.SubnetworkRanges) != (numProjects-1)*d.totalTryCount {
			return fmt.Errorf("the number of subnetwork ranges provided "+
				"should be the same as the number of service projects times the total try count : %d!=%d", len(d.SubnetworkRanges), (numProjects-1)*d.totalTryCount)
		}

		if err := validateSubnetRanges(d.SubnetworkRanges); err != nil {
			return err
		}

		if d.SubnetMode != "" && d.SubnetMode != string(custom) {
			return fmt.Errorf("the subnet-mode must be one of %v for multi-project profile, got: %s", []string{"", string(custom)}, d.SubnetMode)
		}
	} else {
		// Verify for single-project profile
		if d.SubnetMode != "" && d.SubnetMode != string(auto) && d.SubnetMode != string(custom) {
			return fmt.Errorf("--subnet-mode must be one of %v, got: %s", []string{"", string(auto), string(custom)}, d.SubnetMode)
		}
	}

	return d.internalizeNetworkFlags(numProjects)
}

func validateSubnetRanges(subnetworkRanges []string) error {
	// The subnets are passed in a list, each containing groups of 3 CIDR ranges.
	// We need to verify there are no overlaps within the entire group.
	var allSubnetRanges []string
	for _, subnet := range subnetworkRanges {
		ranges := strings.Split(subnet, " ")
		if len(ranges) != 3 {
			return fmt.Errorf("the provided subnetwork range %s is not in the right format, should be like "+
				"10.0.4.0/22 10.0.32.0/20 10.4.0.0/14", subnet)
		}

		allSubnetRanges = append(allSubnetRanges, ranges...)
	}

	if err := assertNoOverlaps(allSubnetRanges); err != nil {
		return fmt.Errorf("error in subnetwork ranges: %v", err)
	}

	return nil
}

func assertNoOverlaps(ranges []string) error {
	var allRanges []*net.IPNet
	for _, rangeString := range ranges {
		_, thisRange, err := net.ParseCIDR(rangeString)
		if err != nil {
			return fmt.Errorf("error parsing %q into a CIDR: %w", rangeString, err)
		}
		for _, previousRange := range allRanges {
			if areOverlapping(thisRange, previousRange) {
				return fmt.Errorf("overlap found within the provided ranges: (%v, %v)", thisRange, previousRange)
			}
		}
		allRanges = append(allRanges, thisRange)
	}
	return nil
}

func areOverlapping(r1, r2 *net.IPNet) bool {
	return r1.Contains(r2.IP) || r2.Contains(r1.IP)
}

func (d *Deployer) internalizeNetworkFlags(numProjects int) error {
	d.subnetworkRangesInternal = make([][]string, d.totalTryCount)
	for tc := 0; tc < d.totalTryCount; tc++ {
		d.subnetworkRangesInternal[tc] = make([]string, numProjects-1)
		for p := 0; p < numProjects-1; p++ {
			index := tc*(numProjects-1) + p
			d.subnetworkRangesInternal[tc][p] = d.SubnetworkRanges[index]
		}
	}

	// Since len(d.PrivateClusterMasterIPRanges) is 0 when private cluster is not requested.
	if d.PrivateClusterAccessLevel == "" {
		return nil
	}

	d.privateClusterMasterIPRangesInternal = make([][]string, d.totalTryCount)
	for tc := 0; tc < d.totalTryCount; tc++ {
		d.privateClusterMasterIPRangesInternal[tc] = make([]string, len(d.Clusters))
		for c := 0; c < len(d.Clusters); c++ {
			index := tc*len(d.Clusters) + c
			d.privateClusterMasterIPRangesInternal[tc][c] = d.PrivateClusterMasterIPRanges[index]
		}
	}
	return nil
}

func (d *Deployer) CreateNetwork() error {
	// Create network if it doesn't exist.
	if runWithNoOutput(exec.Command("gcloud", "compute", "networks", "describe", d.Network,
		"--project="+d.Projects[0],
		"--format=value(name)")) != nil {
		// Assume error implies non-existent.
		subnetMode := d.SubnetMode
		if subnetMode == "" {
			// For single project profile, the subnet-mode could be auto for simplicity.
			// For multiple projects profile, the subnet-mode must be custom and should only be created in the host project.
			//   (Here we consider the first project to be the host project and the rest be service projects)
			//   Reference: https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-shared-vpc#creating_a_network_and_two_subnets
			if len(d.Projects) > 1 {
				subnetMode = string(custom)
			} else {
				subnetMode = string(auto)
			}
		}

		// TODO(chizhg): find a more reliable way to check if the network exists or not.
		klog.V(1).Infof("Couldn't describe network %q, assuming it doesn't exist and creating it", d.Network)
		if err := runWithOutput(exec.Command("gcloud", "compute", "networks", "create", d.Network,
			"--project="+d.Projects[0],
			"--subnet-mode="+subnetMode)); err != nil {
			return err
		}
	}
	return nil
}

func (d *Deployer) CreateSubnets() error {
	// Create subnetworks for the service projects to work with shared VPC if it's a multi-project profile.
	// Reference: https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-shared-vpc#creating_a_network_and_two_subnets
	if len(d.Projects) == 1 {
		return nil
	}
	hostProject := d.Projects[0]
	for i, nr := range d.subnetworkRangesInternal[d.retryCount] {
		serviceProject := d.Projects[i+1]
		parts := strings.Split(nr, " ")
		// The subnetwork name is in the format of `[main_network]-[service_project_id]`.
		subnetName := d.Network + "-" + serviceProject
		createSubnetCommand := []string{
			"gcloud", "compute", "networks", "subnets", "create",
			subnetName,
			"--project=" + hostProject,
			"--region=" + regionFromLocation(d.Regions, d.Zones, d.retryCount),
			"--network=" + d.Network,
			"--range=" + parts[0],
			"--secondary-range",
			fmt.Sprintf("%s-services=%s,%s-pods=%s", subnetName, parts[1], subnetName, parts[2]),
		}
		// Enabling `Private Google Access` on the subnet is needed for private
		// cluster nodes to reach storage.googleapis.com.
		if d.PrivateClusterAccessLevel != "" {
			createSubnetCommand = append(createSubnetCommand, "--enable-private-ip-google-access")
		}
		if err := runWithOutput(exec.Command(createSubnetCommand[0], createSubnetCommand[1:]...)); err != nil {
			return err
		}
	}
	return nil
}

func (d *Deployer) DeleteSubnets(retryCount int) error {
	// Delete the subnetworks if it's a multi-project profile.
	// Reference: https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-shared-vpc#deleting_the_shared_network
	if len(d.Projects) >= 1 {
		hostProject := d.Projects[0]
		for i := 1; i < len(d.Projects); i++ {
			serviceProject := d.Projects[i]
			subnetName := d.Network + "-" + serviceProject
			if err := runWithOutput(exec.Command("gcloud", "compute", "networks", "subnets", "delete",
				subnetName,
				"--project="+hostProject,
				"--region="+regionFromLocation(d.Regions, d.Zones, retryCount),
				"--quiet",
			)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (d *Deployer) DeleteNetwork() error {
	// Do not delete the default network.
	if d.Network == "default" {
		return nil
	}

	return runWithOutput(exec.Command("gcloud", "compute", "networks", "delete", "-q", d.Network,
		"--project="+d.Projects[0], "--quiet"))
}

func transformNetworkName(projects []string, network string) string {
	if len(projects) == 1 {
		return network
	}
	// Multiproject should specify the network at cluster creation such as:
	// projects/HOST_PROJECT_ID/global/networks/SHARED_VPC_NETWORK
	return fmt.Sprintf("projects/%s/global/networks/%s", projects[0], network)
}

// Returns the sub network args needed for the cluster creation command.
// Reference: https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-shared-vpc#creating_a_cluster_in_your_first_service_project
func subNetworkArgs(autopilot bool, projects []string, region, network string, projectIndex int) []string {
	// No sub network args need to be added for creating clusters in the host project.
	if projectIndex == 0 {
		return []string{}
	}

	// subnetwork args are needed for creating clusters in the service project.
	hostProject := projects[0]
	curtProject := projects[projectIndex]
	subnetName := network + "-" + curtProject
	args := []string{
		fmt.Sprintf("--subnetwork=projects/%s/regions/%s/subnetworks/%s", hostProject, region, subnetName),
		fmt.Sprintf("--cluster-secondary-range-name=%s-pods", subnetName),
		fmt.Sprintf("--services-secondary-range-name=%s-services", subnetName),
	}
	// GKE in Autopilot mode does not support --enable-ip-alias flag - https://cloud.google.com/sdk/gcloud/reference/container/clusters/create-auto
	if !autopilot {
		args = append(args, "--enable-ip-alias")
	}
	return args
}

func (d *Deployer) SetupNetwork() error {
	err := enableSharedVPCAndGrantRoles(d.Projects, regionFromLocation(d.Regions, d.Zones, d.retryCount), d.Network)
	if err != nil {
		return err
	}
	return grantHostServiceAgentUserRole(d.Projects)
}

// This function implements https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-shared-vpc#enabling_and_granting_roles
// to enable shared VPC and grant required roles for the multi-project multi-cluster profile.
func enableSharedVPCAndGrantRoles(projects []string, region, network string) error {
	// Nothing needs to be done for single project.
	if len(projects) == 1 {
		return nil
	}

	// The host project will enabled a Shared VPC for other projects and clusters
	// to be part of the same network topology and form a mesh. At current stage,
	// no particular customization has to be made and a single mesh will cover all
	// identified use cases.

	// Enable Shared VPC for multiproject requests on the host project.
	// Assuming we have Shared VPC Admin role at the organization level.
	networkHostProject := projects[0]
	// Shared VPC is still in beta, so we have to use the beta command group here.
	// TODO(chizhg): remove beta after shared VPC is in prod.
	if err := runWithOutput(exec.Command("gcloud", "beta", "compute", "shared-vpc", "enable", networkHostProject)); err != nil {
		// Sometimes we may want to use the projects pre-configured with shared-vpc for testing,
		// and the service account that runs this command might not have the right permission, so do not
		// error out if an error happens here.
		klog.Warningf("Error creating Shared VPC for project %q: %v, it might be due to permission issues.", networkHostProject, err)
	}

	// Associate the rest of the projects.
	for i := 1; i < len(projects); i++ {
		if err := runWithOutput(exec.Command("gcloud", "beta", "compute", "shared-vpc",
			"associated-projects", "add", projects[i],
			"--host-project", networkHostProject)); err != nil {
			klog.Warningf("Error associating project %q to Shared VPC: %v, it might be due to permission issues.", projects[i], err)
		}
	}

	// Grant the required IAM roles to service accounts that belong to the service projects.
	for i := 1; i < len(projects); i++ {
		serviceProject := projects[i]
		subnetName := network + "-" + serviceProject
		// Get the subnet etag.
		subnetETag, err := exec.Output(exec.Command("gcloud", "compute", "networks", "subnets",
			"get-iam-policy", subnetName, "--project="+networkHostProject, "--region="+region, "--format=value(etag)"))
		if err != nil {
			return fmt.Errorf("failed to get the etag for the subnet: %s %s %v", network, region, err)
		}
		// Get the service project number.
		serviceProjectNum, err := getProjectNumber(serviceProject)
		if err != nil {
			return fmt.Errorf("failed to get the project number for %s: %v", serviceProject, err)
		}
		gkeServiceAccount := fmt.Sprintf("service-%s@container-engine-robot.iam.gserviceaccount.com", serviceProjectNum)
		googleAPIServiceAccount := serviceProjectNum + "@cloudservices.gserviceaccount.com"

		// Grant the required IAM roles to service accounts that belong to the service project.
		tempFile, err := os.CreateTemp("", "*.yaml")
		if err != nil {
			return fmt.Errorf("failed to create a temporary yaml file: %v", err)
		}
		policyStr := fmt.Sprintf(networkUserPolicyTemplate, googleAPIServiceAccount, gkeServiceAccount, strings.TrimSpace(string(subnetETag)))
		if err = os.WriteFile(tempFile.Name(), []byte(policyStr), os.ModePerm); err != nil {
			return fmt.Errorf("failed to write the content into %s: %v", tempFile.Name(), err)
		}
		if err = runWithOutput(exec.Command("gcloud", "compute", "networks", "subnets", "set-iam-policy", subnetName,
			tempFile.Name(), "--project="+networkHostProject, "--region="+region)); err != nil {
			return fmt.Errorf("failed to set IAM policy: %v", err)
		}
	}

	return nil
}

// This function implements https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-shared-vpc#grant_host_service_agent_role
// to grant the Host Service Agent User role to each service project's GKE service account.
func grantHostServiceAgentUserRole(projects []string) error {
	// Nothing needs to be done for single project.
	if len(projects) == 1 {
		return nil
	}

	hostProject := projects[0]
	for i := 1; i < len(projects); i++ {
		serviceProject := projects[i]
		serviceProjectNum, err := getProjectNumber(serviceProject)
		if err != nil {
			return err
		}

		gkeServiceAccount := fmt.Sprintf("service-%s@container-engine-robot.iam.gserviceaccount.com", serviceProjectNum)
		if err = runWithOutput(exec.Command("gcloud", "projects", "add-iam-policy-binding", hostProject,
			"--member=serviceAccount:"+gkeServiceAccount,
			"--role=roles/container.hostServiceAgentUser")); err != nil {
			return err
		}
	}
	return nil
}

func (d *Deployer) TeardownNetwork() error {
	err := disableSharedVPCProjects(d.Projects)
	if err != nil {
		return err
	}
	return removeHostServiceAgentUserRole(d.Projects)
}

func disableSharedVPCProjects(projects []string) error {
	// Nothing needs to be done for single project.
	if len(projects) == 1 {
		return nil
	}

	// The host project will enabled a Shared VPC for other projects and clusters
	// to be part of the same network topology and form a mesh. At current stage,
	// no particular customization has to be made and a single mesh will cover all
	// identified use cases

	// Assuming we have Shared VPC Admin role at the organization level
	networkHostProject := projects[0]

	// Disassociate the rest of the projects
	for i := 1; i < len(projects); i++ {
		if err := runWithOutput(exec.Command("gcloud", "beta", "compute", "shared-vpc",
			"associated-projects", "remove", projects[i],
			"--host-project", networkHostProject)); err != nil {
			klog.Warningf("Error removing the associated project %q from Shared VPC: %v", projects[i], err)
		}
	}

	// Disable Shared VPC for multiproject requests on the host project
	if err := runWithOutput(exec.Command("gcloud", "beta", "compute", "shared-vpc", "disable", networkHostProject)); err != nil {
		klog.Warningf("Error disabling Shared VPC for the host project: %v", err)
	}

	return nil
}

// This function implements https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-shared-vpc#removing_the_host_service_agent_user_role
// to remove the Host Service Agent User role granted to each service project's GKE service account.
func removeHostServiceAgentUserRole(projects []string) error {
	// Nothing needs to be done for single project.
	if len(projects) == 1 {
		return nil
	}

	hostProject := projects[0]
	for i := 1; i < len(projects); i++ {
		serviceProject := projects[i]
		serviceProjectNum, err := getProjectNumber(serviceProject)
		if err != nil {
			return err
		}

		gkeServiceAccount := fmt.Sprintf("service-%s@container-engine-robot.iam.gserviceaccount.com", serviceProjectNum)
		if err = runWithOutput(exec.Command("gcloud", "projects", "remove-iam-policy-binding", hostProject,
			"--member=serviceAccount:"+gkeServiceAccount,
			"--role=roles/container.hostServiceAgentUser")); err != nil {
			return err
		}
	}
	return nil
}

// This function returns the args required for creating a private cluster.
// Reference: https://cloud.google.com/kubernetes-engine/docs/how-to/private-clusters#top_of_page
func getPrivateClusterArgs(projects []string, network, accessLevel string, masterIPRanges []string, clusterInfo cluster, autopilot bool) []string {
	common := []string{
		"--enable-private-nodes",
	}
	// GKE in Autopilot mode does not support certain flags - https://cloud.google.com/kubernetes-engine/docs/how-to/private-clusters
	if !autopilot {
		common = append(common,
			"--enable-ip-alias",
			"--no-enable-basic-auth",
			"--master-ipv4-cidr="+masterIPRanges[clusterInfo.index],
			"--no-issue-client-certificate")
	}

	// For multi-project profile, it'll be using the shared vpc, which creates subnets before cluster creation.
	// So only create subnetworks if it's single-project profile.
	if len(projects) == 1 {
		subnetName := network + "-" + clusterInfo.name
		common = append(common, "--create-subnetwork=name="+subnetName)
	}

	switch accessLevel {
	case string(no):
		common = append(common, "--enable-master-authorized-networks",
			"--enable-private-endpoint")
	case string(limited):
		common = append(common, "--enable-master-authorized-networks")
	case string(unrestricted):
		common = append(common, "--no-enable-master-authorized-networks")
	}

	return common
}
