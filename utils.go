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
	"encoding/json"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/service/ec2"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

func getInstanceIdentity() (*awsIdentity, error) {
	// step: retrieve the dynamic instance document
	res, err := http.Get("http://169.254.169.254/latest/dynamic/instance-identity/document")
	if err != nil {
		return nil, err
	}

	// step: read in the response body
	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	// step: decode the response
	instance := new(awsIdentity)
	if err := json.Unmarshal([]byte(content), instance); err != nil {
		return nil, err
	}

	hostname, err := getMetaLocalHostname()
	if err != nil {
		return nil, err
	}
	instance.PrivateDNSName = hostname

	return instance, nil
}

// getEtcdEndpoints constructs a list of endpoints from a list of aws instances
func getEtcdEndpoints(instances []*ec2.Instance) []string {
	var list []string
	for _, i := range instances {
		location := *i.PrivateDnsName
		if config.privateIPs {
			location = *i.PrivateIpAddress
		}
		list = append(list, fmt.Sprintf("%s://%s:%d", config.etcdClientScheme, location, config.etcdClientPort))
	}

	return list
}

func writeFile(filename, content string) error {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0444)
	if err != nil {
		return err
	}

	if _, err := file.WriteString(content); err != nil {
		return err
	}

	return nil
}

func getPeerURLs(members []*ec2.Instance) string {
	var list []string
	for _, i := range members {
		node := *i.PrivateDnsName
		if config.privateIPs {
			node = *i.PrivateIpAddress
		}
		list = append(list, fmt.Sprintf("%s=%s", *i.InstanceId, getPeerURL(node)))
	}

	return strings.Join(list, ",")
}

// getPeerURL construct the peer url for the cluster member
func getPeerURL(location string) string {
	return fmt.Sprintf("%s://%s:%d", config.etcdPeerScheme, location, config.etcdPeerPort)
}

// getMetaLocalHostname retrieves the dns hostname from the metadata service
func getMetaLocalHostname() (string, error) {
	// step: retrieve the dynamic instance document
	res, err := http.Get("http://169.254.169.254/latest/meta-data/local-hostname")
	if err != nil {
		return "", err
	}

	// step: read in the response body
	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// isScheme checks the scheme is valid
func isSchema(s string) bool {
	if s == "https" || s == "http" {
		return true
	}

	return false
}

// isPort checks the port is valid
func isPort(p int) bool {
	if p >= 1 && p <= 65534 {
		return true
	}

	return false
}

func printUsage(message string) {
	flag.Usage()
	fmt.Fprintf(os.Stderr, "\n[error] %s\n", message)
	os.Exit(1)
}
