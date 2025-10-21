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

	"k8s.io/klog/v2"
	"sigs.k8s.io/kubetest2/pkg/exec"
)

func (d *Deployer) EnsureNat() error {
	if !d.CreateNat {
		return nil
	}
	if d.Network == "default" {
		return fmt.Errorf("NAT router should be set manually for the default network")
	}
	region := regionFromLocation(d.Regions, d.Zones, d.retryCount)
	nat := d.getNatName()
	hostProject := d.Projects[0]

	if runWithNoOutput(exec.Command("gcloud", "compute", "routers", "describe", nat,
		"--project="+hostProject,
		"--region="+region,
		"--format=value(name)")) != nil {
		klog.V(1).Infof("Couldn't describe router '%s', assuming it doesn't exist and creating it", nat)
		if err := runWithOutput(exec.Command("gcloud", "compute", "routers", "create", nat,
			"--project="+hostProject,
			"--network="+d.Network,
			"--region="+region)); err != nil {
			return fmt.Errorf("error creating NAT router: %w", err)
		}
	}
	// Create this unique NAT configuration only if it does not exist yet.
	if runWithNoOutput(exec.Command("gcloud", "compute", "routers", "nats", "describe", nat,
		"--project="+hostProject,
		"--router="+nat,
		"--region="+region,
		"--format=value(name)")) != nil {
		klog.V(1).Infof("Couldn't describe NAT '%s', assuming it doesn't exist and creating it", nat)
		if err := runWithOutput(exec.Command("gcloud", "compute", "routers", "nats", "create", nat,
			"--project="+hostProject,
			"--router="+nat,
			"--region="+region,
			"--auto-allocate-nat-external-ips",
			"--nat-primary-subnet-ip-ranges")); err != nil {
			return fmt.Errorf("error adding NAT to a router: %w", err)
		}
	}

	return nil
}

func (d *Deployer) getNatName() string {
	return "nat-router-" + d.Clusters[0]
}

func (d *Deployer) CleanupNat() error {
	if !d.CreateNat {
		return nil
	}
	region := regionFromLocation(d.Regions, d.Zones, d.retryCount)
	nat := d.getNatName()
	hostProject := d.Projects[0]

	// Delete NAT router. That will remove NAT configuration as well.
	if runWithNoOutput(exec.Command("gcloud", "compute", "routers", "describe", nat,
		"--project="+hostProject,
		"--region="+region,
		"--format=value(name)")) == nil {
		klog.V(1).Infof("Found NAT router '%s', deleting", nat)
		err := runWithOutput(exec.Command("gcloud", "compute", "routers", "delete", "-q", nat,
			"--project="+hostProject,
			"--region="+region))
		if err != nil {
			return fmt.Errorf("error deleting NAT router: %w", err)
		}
	} else {
		klog.V(1).Infof("Found no NAT router '%s', assuming resources are clean", nat)
	}

	return nil
}
