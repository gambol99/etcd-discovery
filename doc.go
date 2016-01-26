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
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const (
	program = "etcd-discovery"
	author  = "Rohith"
	email   = "gambol99@gmail.com"
	version = "v0.0.1"
)

// awsClient is the wrapper for aws api access
type awsClient struct {
	// the client for auto-scaling
	asg *autoscaling.AutoScaling
	// the instance client
	compute *ec2.EC2
}

// awsIdentity is the instance document for the running instance
type awsIdentity struct {
	// InstanceID is the id of the instance
	InstanceID string `json:"instanceId"`
	// Region is the aws region we in
	Region string `json:"region"`
	// LocalIP is the local ip address of the instance
	LocalIP string `json:"privateIp"`
	// PrivateDNSName
	PrivateDNSName string
	// ImageID is the ami we were built from
	ImageID string `json:"imageId"`
	// InstanceType is the build
	InstanceType string `json:"instanceType"`
	// AvailabilityZone is the AZ the instance is on
	AvailabilityZone string `json:"availabilityZone"`
}
