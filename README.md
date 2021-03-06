### **Etcd Discovery**
----

The Etcd Discovery service is small utility for running a [Etcd](https://github.com/coreos/etcd) cluster within a AWS auto-scaling group. The service provides the means to add, remove and discovery instances in the auto-scaling group and to reconcile the cluster when things change. 

##### **Building**

A [Makefile](https://github.com/gambol99/etcd-discovery/blob/master/Makefile) exists in the source root, so assuming you've Go and make; running *make* should suffice. Alternatively you can build the project inside a golang container by typing **make docker-build**

#### **Configuration**
----

```shell
[jest@starfury etcd-discovery]$ bin/etcd-discovery -h
Usage of bin/etcd-discovery:
  -alsologtostderr
    	log to standard error as well as files
  -environment-file string
    	the file to write the etcd environment variables
  -etcd-client-port int
    	is the port the etcd client should be listening on (default 2379)
  -etcd-client-schema string
    	is the protocol schema we should use for client connections (default "https")
  -etcd-peer-port int
    	is the port the etcd peer should be listening on (default 2380)
  -etcd-peer-scheme string
    	is the protocol schema we should use for etcd peer connections (default "https")
  -log_backtrace_at value
    	when logging hits line file:N, emit a stack trace (default :0)
  -log_dir string
    	If non-empty, write log files in this directory
  -logtostderr
    	log to standard error instead of files
  -private-addresses
    	add the etcd peers using their ip addresses rather than domain names
  -private-hostnames
    	add the etcd peers using the dns names rather than up addresses (default true)
  -proxy-mode
    	whether or not we are operating in etcd proxy mode
  -scaling-group-name string
    	is the name of the aws auto-scaling group which has the etcd masters
  -stderrthreshold value
    	logs at or above this threshold go to stderr
  -v value
    	log level for V logs
  -vmodule value
    	comma-separated list of pattern=N settings for file-filtered logging
```

#### **Example Usage**

Lets assume you have two auto-scaling groups, the etcd cluster and another cluster whom are proxy-mode only node i.e. consumers. Taken from the cloudinit userdata (CoreOS), systemd unit could like like

```YAML
# Download the etcd-discovery binary
- name: install-etcd-discovery.service
  command: start
  content: |
    [Unit]
    Description=Etcd Discovery Service
    Before=etcd2.service

    [Service]
    RemainAfterExit=yes
    Restart=always
    RestartSec=10
    Environment="URL=${etcd_discovery_url}"
    Environment="FILENAME=/opt/bin/etcd-discovery"
    Environment="FILENAME_GZ=/opt/bin/etcd-discovery.gz"
    Environment="MD5=${etcd_discovery_md5}"
    ExecStartPre=/usr/bin/mkdir -p /opt/bin /etc/sysconfig
    ExecStartPre=-/usr/bin/rm -f /opt/bin/etcd-discovery
    ExecStart=/usr/bin/bash -c 'until [[ -x $${FILENAME} ]] && [[ $(md5sum $${FILENAME} | cut -f1 -d" ") == $${MD5} ]]; \
      do wget -q -O $${FILENAME_GZ} $${URL} && gunzip $${FILENAME_GZ} && chmod +x $${FILENAME}; done'
# Start the etcd-discovery (running in master mode), find the nodes in the cluster and template
# out peer configuration to /etc/sysconfig/etcd-discovery file
- name: etcd-discovery.service
  command: start
  content: |
    [Unit]
    Description=Etcd Discovery Service
    Requires=install-etcd-discovery.service
    After=install-etcd-discovery.service
    Before=etcd2.service

    [Service]
    RemainAfterExit=yes
    Restart=always
    RestartSec=10
    ExecStart=/opt/bin/etcd-discovery -environment-file=/etc/sysconfig/etcd-discovery -logtostderr -v=10
- name: etcd2.service
  command: start
  enable: true
  drop-ins:
  # read in the peer configuration generated by etcd-discovery service    
  - name: 30-etcd_peers.conf
    content: |
      [Service]
      EnvironmentFile=/etc/sysconfig/etcd-discovery
  - name: 12-advertized.conf
    content: |
      [Service]
      Environment="ETCD_ADVERTISE_CLIENT_URLS=https://%H:2379"
      Environment="ETCD_INITIAL_ADVERTISE_PEER_URLS=https://%H:2380"
```

The logic for this is fairly simple

  - Get the auto-scaling group my instance is in
  - Get all the instances from the auto-scaling group
  - Check to see if the cluster has formed yet
    - Check for any terminated instances in the cluster, check if theses instances are referenced in the etcd cluster and remove them
    - Check with etcd that my instances is a cluster member and if not, add myself as a member
  - Template out the peer configuration to the file

For the consumer nodes we can use something similar to the below.

```YAML
- name: install-etcd-discovery.service
  command: start
  content: |
    [Unit]
    Description=Etcd Discovery Service
    Before=etcd2.service

    [Service]
    RemainAfterExit=yes
    Restart=always
    RestartSec=10
    Environment="URL=${etcd_discovery_url}"
    Environment="FILENAME=/opt/bin/etcd-discovery"
    Environment="FILENAME_GZ=/opt/bin/etcd-discovery.gz"
    Environment="MD5=${etcd_discovery_md5}"
    ExecStartPre=/usr/bin/mkdir -p /opt/bin /etc/sysconfig
    ExecStartPre=-/usr/bin/rm -f /opt/bin/etcd-discovery
    ExecStart=/usr/bin/bash -c 'until [[ -x $${FILENAME} ]] && [[ $(md5sum $${FILENAME} | cut -f1 -d" ") == $${MD5} ]]; \
      do wget -q -O $${FILENAME_GZ} $${URL} && gunzip $${FILENAME_GZ} && chmod +x $${FILENAME}; done'
- name: etcd-discovery.service
  command: start
  content: |
    [Unit]
    Description=Etcd Discovery Service
    Requires=install-etcd-discovery.service
    After=install-etcd-discovery.service
    Before=etcd2.service

    [Service]
    RemainAfterExit=yes
    Restart=always
    RestartSec=10
    ExecStart=/opt/bin/etcd-discovery \
      -scaling-group-name=THE_NAME_OF_ETCD_SCALING_GROUP \
      -proxy-mode=true \
      -environment-file=/etc/sysconfig/etcd-discovery \
      -logtostderr \
      -v=10
- name: etcd2.service
  command: start
  enable: true
  drop-ins:
  - name: 30-etcd_peers.conf
    content: |
      [Service]
      EnvironmentFile=/etc/sysconfig/etcd-discovery
```
