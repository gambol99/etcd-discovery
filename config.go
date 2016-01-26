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
	"flag"
	"fmt"
)

// discoveryConfig is the configuration for the service
type discoveryConfig struct {
	// environmentFile is the file to write the environment variables
	environmentFile string
	// etcdPeerScheme is the protocol for the peers
	etcdPeerScheme string
	// clientScheme is the protocol for the clients
	etcdClientScheme string
	// etcdClientPort is the port for the clients
	etcdClientPort int
	// etcdPeerPort is the port for the peers
	etcdPeerPort int
	// we should use private ip addresses for the etcd peers
	privateIPs bool
	// we should use private dns names for the etcd peers
	privateHostnames bool
	// we are running a etcd service in proxy mode
	proxyMode bool
	// groupName is the name of the autoscaling group with the etcd masters
	groupName string
}

var config *discoveryConfig

func init() {
	config = new(discoveryConfig)
	flag.StringVar(&config.environmentFile, "environment-file", "", "the file to write the etcd environment variables")
	flag.StringVar(&config.etcdPeerScheme, "etcd-peer-scheme", "https", "is the protocol schema we should use for etcd peer connections")
	flag.StringVar(&config.etcdClientScheme, "etcd-client-schema", "https", "is the protocol schema we should use for client connections")
	flag.IntVar(&config.etcdClientPort, "etcd-client-port", 2379, "is the port the etcd client should be listening on")
	flag.IntVar(&config.etcdPeerPort, "etcd-peer-port", 2380, "is the port the etcd peer should be listening on")
	flag.StringVar(&config.groupName, "scaling-group-name", "", "is the name of the aws auto-scaling group which has the etcd masters")
	flag.BoolVar(&config.privateIPs, "private-addresses", false, "add the etcd peers using their ip addresses rather than domain names")
	flag.BoolVar(&config.privateHostnames, "private-hostnames", true, "add the etcd peers using the dns names rather than up addresses")
	flag.BoolVar(&config.proxyMode, "proxy-mode", false, "whether or not we are operating in etcd proxy mode")
}

// getConfig grab the command line options, validate the configuration and returns
func getConfig() error {
	flag.Parse()

	if config.environmentFile == "" {
		return fmt.Errorf("you have not set the environment file path to write to")
	}
	if !isPort(config.etcdPeerPort) {
		return fmt.Errorf("etcd peer port %d is an invalid port", config.etcdPeerPort)
	}
	if !isPort(config.etcdClientPort) {
		return fmt.Errorf("etcd client port %d is an invalid port", config.etcdClientPort)
	}
	if config.proxyMode && config.groupName == "" {
		return fmt.Errorf("you must set the autoscaling group name when in proxy mode")
	}
	if config.privateIPs && config.privateHostnames {
		return fmt.Errorf("you cannot have both private address and hostnames enabled")
	}
	if !isSchema(config.etcdClientScheme) {
		return fmt.Errorf("the scheme %s for etcd client is invalid", config.etcdClientScheme)
	}
	if !isSchema(config.etcdPeerScheme) {
		return fmt.Errorf("the scheme %s for etcd peer is invalid", config.etcdPeerScheme)
	}

	return nil
}
