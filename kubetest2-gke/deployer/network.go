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
	"io/ioutil"
	"os"
	"strings"

	"k8s.io/klog"

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

func (d *deployer) verifyNetworkFlags() error {
	// For single project, no verification is needed.
	numProjects := len(d.projects)
	if numProjects == 0 {
		numProjects = d.boskosProjectsRequested
	}
	if numProjects == 1 {
		return nil
	}

	if d.network == "default" {
		return errors.New("the default network cannot be used for multi-project profile")
	}

	if len(d.subnetworkRanges) != numProjects-1 {
		return fmt.Errorf("the number of subnetwork ranges provided "+
			"should be the same as the number of service projects: %d!=%d", len(d.subnetworkRanges), numProjects-1)
	}

	for _, sr := range d.subnetworkRanges {
		parts := strings.Split(sr, " ")
		if len(parts) != 3 {
			return fmt.Errorf("the provided subnetwork range %s is not in the right format, should be like "+
				"10.0.4.0/22 10.0.32.0/20 10.4.0.0/14", sr)
		}
	}

	if d.privateClusterAccessLevel != "" && d.privateClusterAccessLevel != string(no) &&
		d.privateClusterAccessLevel != string(limited) && d.privateClusterAccessLevel != string(unrestricted) {
		return fmt.Errorf("--private-cluster-access-level must be one of %v", []string{"", string(no), string(limited), string(unrestricted)})
	}
	if d.privateClusterAccessLevel != "" && d.privateClusterMasterIPRange == "" {
		return fmt.Errorf("--private-cluster-master-ip-range must not be empty when requesting a private cluster")
	}

	return nil
}

func (d *deployer) createNetwork() error {
	// Create network if it doesn't exist.
	// For single project profile, the subnet-mode could be auto for simplicity.
	// For multiple projects profile, the subnet-mode must be custom and should only be created in the host project.
	//   (Here we consider the first project to be the host project and the rest be service projects)
	//   Reference: https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-shared-vpc#creating_a_network_and_two_subnets
	subnetMode := "auto"
	if len(d.projects) > 1 {
		subnetMode = "custom"
	}
	if runWithNoOutput(exec.Command("gcloud", "compute", "networks", "describe", d.network,
		"--project="+d.projects[0],
		"--format=value(name)")) != nil {
		// Assume error implies non-existent.
		// TODO(chizhg): find a more reliable way to check if the network exists or not.
		klog.V(1).Infof("Couldn't describe network %q, assuming it doesn't exist and creating it", d.network)
		if err := runWithOutput(exec.Command("gcloud", "compute", "networks", "create", d.network,
			"--project="+d.projects[0],
			"--subnet-mode="+subnetMode)); err != nil {
			return err
		}
	}

	// Create subnetworks for the service projects to work with shared VPC if it's a multi-project profile.
	// Reference: https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-shared-vpc#creating_a_network_and_two_subnets
	if len(d.projects) == 1 {
		return nil
	}
	hostProject := d.projects[0]
	for i, nr := range d.subnetworkRanges {
		serviceProject := d.projects[i+1]
		parts := strings.Split(nr, " ")
		// The subnetwork name is in the format of `[main_network]-[service_project_id]`.
		subnetName := d.network + "-" + serviceProject
		if err := runWithOutput(exec.Command("gcloud", "compute", "networks", "subnets", "create",
			subnetName,
			"--project="+hostProject,
			"--region="+d.region,
			"--network="+d.network,
			"--range="+parts[0],
			"--secondary-range",
			fmt.Sprintf("%s-services=%s,%s-pods=%s", subnetName, parts[1], subnetName, parts[2]),
		)); err != nil {
			return err
		}
	}

	return nil
}

func (d *deployer) deleteNetwork() error {
	// Do not delete the default network.
	if d.network == "default" {
		return nil
	}

	// Delete the subnetworks if it's a multi-project profile.
	// Reference: https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-shared-vpc#deleting_the_shared_network
	if len(d.projects) >= 1 {
		hostProject := d.projects[0]
		for i := 1; i < len(d.projects); i++ {
			serviceProject := d.projects[i]
			subnetName := d.network + "-" + serviceProject
			if err := runWithOutput(exec.Command("gcloud", "compute", "networks", "subnets", "delete",
				subnetName,
				"--project="+hostProject,
				"--region="+d.region,
				"--quiet",
			)); err != nil {
				return err
			}
		}
	}

	if err := runWithOutput(exec.Command("gcloud", "compute", "networks", "delete", "-q", d.network,
		"--project="+d.projects[0], "--quiet")); err != nil {
		return err
	}

	return nil
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
func subNetworkArgs(projects []string, region, network string, projectIndex int) []string {
	// No sub network args need to be added for creating clusters in the host project.
	if projectIndex == 0 {
		return []string{}
	}

	// subnetwork args are needed for creating clusters in the service project.
	hostProject := projects[0]
	curtProject := projects[projectIndex]
	subnetName := network + "-" + curtProject
	return []string{
		"--enable-ip-alias",
		fmt.Sprintf("--subnetwork=projects/%s/regions/%s/subnetworks/%s", hostProject, region, subnetName),
		fmt.Sprintf("--cluster-secondary-range-name=%s-pods", subnetName),
		fmt.Sprintf("--services-secondary-range-name=%s-services", subnetName),
	}
}

func (d *deployer) setupNetwork() error {
	if err := enableSharedVPCAndGrantRoles(d.projects, d.region, d.network); err != nil {
		return err
	}
	if err := grantHostServiceAgentUserRole(d.projects); err != nil {
		return err
	}
	return nil
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
		tempFile, err := ioutil.TempFile("", "*.yaml")
		if err != nil {
			return fmt.Errorf("failed to create a temporary yaml file: %v", err)
		}
		policyStr := fmt.Sprintf(networkUserPolicyTemplate, googleAPIServiceAccount, gkeServiceAccount, strings.TrimSpace(string(subnetETag)))
		if err = ioutil.WriteFile(tempFile.Name(), []byte(policyStr), os.ModePerm); err != nil {
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

func (d *deployer) teardownNetwork() error {
	if err := disableSharedVPCProjects(d.projects); err != nil {
		return err
	}
	if err := removeHostServiceAgentUserRole(d.projects); err != nil {
		return err
	}
	return nil
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
func privateClusterArgs(network, cluster, accessLevel, masterIPRange string) []string {
	if accessLevel == "" {
		return []string{}
	}

	subnetName := network + "-" + cluster
	common := []string{
		"--create-subnetwork name=" + subnetName,
		"--enable-ip-alias",
		"--enable-private-nodes",
		"--no-enable-basic-auth",
		"--master-ipv4-cidr=" + masterIPRange,
		"--no-issue-client-certificate",
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
