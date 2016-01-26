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
	"strings"

	etcd "github.com/coreos/etcd/client"
	"github.com/golang/glog"
	"golang.org/x/net/context"
)

type etcdClient struct {
	// the etcd client
	c etcd.Client
	// the members client
	client etcd.MembersAPI
}

// newEtcdClient create a new etcd client wrapper
func newEtcdClient(endpoints []string) (*etcdClient, error) {
	glog.V(3).Infof("creating a new etcd client, endpoints: %s", strings.Join(endpoints, ","))
	// step: create a client for etcd
	c, err := etcd.New(etcd.Config{Endpoints: endpoints})
	if err != nil {
		return nil, err
	}

	return &etcdClient{
		c:      c,
		client: etcd.NewMembersAPI(c),
	}, nil
}

// listMembers retrieves a list of members
func (r *etcdClient) listMembers() ([]etcd.Member, error) {
	members, err := r.client.List(context.Background())
	if err != nil {
		return nil, r.handleError(err)
	}

	return members, nil
}

// addMember add the member to the cluster
func (r *etcdClient) addMember(name, url string) error {
	if found, err := r.hasMember(name); err != nil {
		return err
	} else if found {
		return nil
	}

	if _, err := r.client.Add(context.Background(), url); err != nil {
		return r.handleError(err)
	}

	return nil
}

// deleteMemeber remove's a member from the cluster
func (r *etcdClient) deleteMember(id string) error {
	// step: delete the member
	if err := r.client.Remove(context.Background(), id); err != nil {
		return r.handleError(err)
	}

	return nil
}

// getMember retrieves a specific member from the cluster
func (r *etcdClient) getMember(name string) (etcd.Member, error) {
	members, err := r.listMembers()
	if err != nil {
		return etcd.Member{}, err
	}

	for _, m := range members {
		if m.Name == name {
			return m, nil
		}
	}

	return etcd.Member{}, fmt.Errorf("the member does not exist")
}

// hasMember checks if a member exists
func (r *etcdClient) hasMember(name string) (bool, error) {
	members, err := r.listMembers()
	if err != nil {
		return false, err
	}
	for _, m := range members {
		if m.Name == name {
			return true, nil
		}
	}

	return false, nil
}

func (r *etcdClient) handleError(err error) error {
	if err == context.Canceled {
		glog.Errorf("the operation was canceled")
	} else if err == context.DeadlineExceeded {
		glog.Errorf("the operation exceeded deadline")
	} else if cerr, ok := err.(*etcd.ClusterError); ok {
		glog.Errorf("cluster error, %s", cerr.Detail())
	} else {
		glog.Errorf("bad cluster configuration")
	}

	return err
}
