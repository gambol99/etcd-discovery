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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/golang/glog"
)

func newAwsClient(region string) (*awsClient, error) {
	glog.V(3).Infof("creating a aws client, region: %s", region)

	// step: get the auto-scaling client
	asg := autoscaling.New(session.New(), &aws.Config{
		Region: aws.String(region),
	})
	// step: get the ec2 instance client
	compute := ec2.New(session.New(), &aws.Config{
		Region: aws.String(region),
	})

	return &awsClient{
		asg:     asg,
		compute: compute,
	}, nil
}

// getAutoScalingGroupWithInstanceID finds the auto-scaling group the instance is in
func (r *awsClient) getAutoScalingGroupWithInstanceID(id string) (string, error) {
	glog.V(10).Infof("searching for autoscaling group with instance id: %s", id)

	// step: iterate the auto-scaling groups
	groups, err := r.asg.DescribeAutoScalingGroups(nil)
	if err != nil {
		return "", err
	}
	glog.V(10).Infof("found %d auto-scaling groups in region", len(groups.AutoScalingGroups))

	// step: find the group which has the instance id
	for _, g := range groups.AutoScalingGroups {
		glog.V(10).Infof("checking scaling group: %s for instance id: %s", *g.AutoScalingGroupName, id)
		for _, i := range g.Instances {
			if *i.InstanceId == id {
				glog.V(3).Infof("found instance id: %s in group: %s", id, *g.AutoScalingGroupName)
				return *g.AutoScalingGroupName, nil
			}
		}
	}

	return "", fmt.Errorf("no auto-scaling group found with instance id: %s", id)
}

func (r awsClient) isInstanceTerminated(id string) (bool, error) {
	glog.V(10).Infof("checking if instance id: %s is terminated", id)

	instance, err := r.compute.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("instance-id"),
				Values: []*string{aws.String(id)},
			},
		},
	})
	if err != nil {
		return false, err
	}

	if len(instance.Reservations) <= 0 {
		glog.Warningf("no instance %s found in the region", id)
		return false, nil
	}

	status := *instance.Reservations[0].Instances[0].State.Name
	glog.V(3).Infof("instance status of %s: %s", id, status)

	if status != "terminated" {
		return false, nil
	}

	return true, nil
}

func (r *awsClient) getAutoScalingInstances(name string) ([]*ec2.Instance, error) {
	glog.V(10).Infof("retrieving the instance from auto-scaling group: %s", name)
	var list []*ec2.Instance

	// step: find the auto-scaling group
	group, err := r.getAutoScalingGroupByName(name)
	if err != nil {
		return list, err
	}

	glog.V(4).Infof("found %d instances in group: %s", len(group.Instances), name)

	// step: iterate the auto-scaling instance id's
	for _, i := range group.Instances {
		glog.V(5).Infof("group: %s, instance: %s, status: %s", name, *i.InstanceId, *i.HealthStatus)

		if *i.HealthStatus != "Healthy" {
			glog.Warningf("the instance %s, status: %s skipping as member", *i.InstanceId, *i.HealthStatus)
			continue
		}

		// step: grab the instance details
		instance, err := r.compute.DescribeInstances(&ec2.DescribeInstancesInput{
			Filters: []*ec2.Filter{
				{
					Name:   aws.String("instance-id"),
					Values: []*string{i.InstanceId},
				},
			},
		})
		if err != nil {
			return nil, err
		}
		if len(instance.Reservations[0].Instances) <= 0 {
			glog.Warningf("the instance id: %s was not found", *i.InstanceId)
			continue
		}

		// step: bypass any instances which are not running
		i := instance.Reservations[0].Instances[0]
		glog.V(10).Infof("instance: %s, status: %s", *i.InstanceId, *i.State.Name)
		if *i.State.Name != "running" {
			glog.Warningf("skipping instance: %s as the status is not running", *i.InstanceId)
			continue
		}

		list = append(list, i)
	}

	return list, nil
}

func (r *awsClient) getAutoScalingGroupByName(name string) (*autoscaling.Group, error) {
	groups, err := r.asg.DescribeAutoScalingGroups(nil)
	if err != nil {
		return nil, err
	}
	for _, gp := range groups.AutoScalingGroups {
		if *gp.AutoScalingGroupName == name {
			return gp, nil
		}
	}

	return nil, fmt.Errorf("no auto-scaling group %s found", name)
}
