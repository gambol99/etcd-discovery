/*
Copyright 2015 All rights reserved.
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

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/golang/glog"
)

var (
	// the aws client
	awsCli *awsClient
)

//
// Steps:
//  - grab the command line configuration
//  - retrieve the instance identity document from metadata service
//  - find the instances in the auto-scaling group
//  - create a etcd client from the instance and see if we can connect the cluster
//  - write out the environment file
//  - if in proxy mode we can exit here
//  - check if the instance id exists in the cluster and if not, try to add us
//  - list the members in the cluster and find the instance status, if terminated, try and remove
//

func main() {
	if err := getConfig(); err != nil {
		printUsage(fmt.Sprintf("invalid configuration, error: %s", err))
	}
	glog.Infof("starting %s version: %s, author: %s <%s>", program, version, author, email)

	// step: retrieve this instances identity
	identity, err := getInstanceIdentity()
	if err != nil {
		glog.Errorf("failed to get the instance identity, error: %s", err)
		os.Exit(1)
	}

	// step: create a aws client
	awsCli, err = newAwsClient(identity.Region)
	if err != nil {
		glog.Errorf("failed to create a aws client, error: %s", err)
		os.Exit(1)
	}

	// step: get a list of instances in the scaling group
	instances, err := getAutoScalingMembers(identity)
	if err != nil {
		glog.Errorf("failed to retrieve a list of instance from auto-scaling group, error: %s", err)
		os.Exit(1)
	}

	cluster_state := "new"

	// step: create an etcd client for us
	client, err := newEtcdClient(getEtcdEndpoints(instances))
	if err != nil {
		glog.Warningf("failed to create an etcd client, error: %s", err)
	} else {
		if _, err := client.listMembers(); err == nil {
			cluster_state = "existing"
		}
	}

	if config.proxyMode {
		cluster_state = "existing"
	}

	// step: write out the environment file
	glog.Infof("writing the environment variables to file: %s", config.environmentFile)
	if err := writeEnvironment(config.environmentFile, identity, instances, cluster_state, config.proxyMode); err != nil {
		glog.Errorf("failed to write the environment file, error: %s", err)
		os.Exit(1)
	}

	// step: create an etcd client from the members if NOT in proxy mode
	if !config.proxyMode {
		glog.Infof("attempting to add the member: %s into the cluster", identity.InstanceID)
		// step: update the etcd cluster
		if err := syncMembership(identity, getEtcdEndpoints(instances)); err != nil {
			glog.Errorf("failed to update the etcd cluster, error: %s", err)
			os.Exit(1)
		}
	}
}

// getAutoScalingMembers retrieve the members from the auto-scaling group
func getAutoScalingMembers(identity *awsIdentity) ([]*ec2.Instance, error) {
	autoScalingGroupName := config.groupName
	// step: are we in proxy mode?
	if autoScalingGroupName == "" {
		glog.Infof("etcd auto-scaling group not set, using instance id %s for search", identity.InstanceID)
		name, err := awsCli.getAutoScalingGroupWithInstanceID(identity.InstanceID)
		if err != nil {
			return nil, err
		}
		autoScalingGroupName = name
	}

	glog.Infof("retrieving the instances from the group: %s", autoScalingGroupName)
	// step: get a list of the instance
	instances, err := awsCli.getAutoScalingInstances(autoScalingGroupName)
	if err != nil {
		return nil, err
	}

	return instances, nil
}

// syncMembership is responsible for adding the new member into the cluster and cleaning up anyone
// that doesn't need to be there anymore
func syncMembership(identity *awsIdentity, instances []string) error {
	memberID := identity.InstanceID
	client, err := newEtcdClient(instances)
	if err != nil {
		return err
	}

	glog.Infof("retrievig a list of cluster members")
	members, err := client.listMembers()
	if err != nil {
		return err
	}

	glog.Infof("found %d members in the cluster", len(members))

	// step: attempt to remove any boxes from the cluster which have terminated
	glog.Infof("checking if any cluster members can been cleaned out")

	// step: remove any members no longer required
	for _, i := range members {
		glog.V(10).Infof("checking if instance: %s, url: %s is still alive", i.Name, i.PeerURLs)
		terminated, err := awsCli.isInstanceTerminated(i.Name)
		if err != nil {
			glog.Warningf("failed to determine if member %s is running, error: %s", i.Name, err)
			continue
		}
		if terminated {
			glog.Infof("member %s has been terminated, removing from the cluster", i.Name)
			removed := false
			for j := 0; j < 3; j++ {
				if err := client.deleteMember(i.ID); err != nil {
					glog.Errorf("failed to remove the member %s, error: %s", i.Name, err)
					<-time.After(time.Duration(3) * time.Second)
				} else {
					glog.Infof("successfully remove the member: %s", i.Name)
					removed = true
					break
				}
			}

			if !removed {
				return fmt.Errorf("failed to remove the member: %s", i.Name)
			}
		}
	}

	// step: if we are not in the cluster, attempt to add our self
	glog.V(3).Infof("checking the member %s is part of the cluster", memberID)
	if found, err := client.hasMember(memberID); err != nil {
		return err
	} else if !found {
		glog.Infof("member %s is not presently part of the cluster, adding now", memberID)

		nodeAddress := identity.PrivateDNSName
		if config.privateIPs {
			nodeAddress = identity.LocalIP
		}
		peerURL := getPeerURL(nodeAddress)

		glog.Infof("attempting to add the member, peerURL: %s", peerURL)

		if err := client.addMember(memberID, peerURL); err != nil {
			glog.Errorf("failed to add the member into the cluster, error: %s", err)
		}
		glog.Infof("successfully added the member: %s to cluster", memberID)
	} else {
		glog.Infof("member %s is already in the cluster, moving to cleanup", memberID)
	}

	return nil
}

func writeEnvironment(filename string, identity *awsIdentity, members []*ec2.Instance, state string, proxy bool) error {
	// step: generate the cluster url
	peersURL := getPeerURLs(members)
	mode := "off"
	if proxy {
		mode = "on"
	}
	content := fmt.Sprintf(`
ETCD_INITIAL_CLUSTER_STATE=%s
ETCD_NAME=%s
ETCD_INITIAL_CLUSTER="%s"
ETCD_PROXY="%s"
`, state, identity.InstanceID, peersURL, mode)

	if err := writeFile(filename, content); err != nil {
		return err
	}

	return nil
}
