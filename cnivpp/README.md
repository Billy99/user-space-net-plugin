# VPP CNI Library Intro
VPP CNI Library is written in GO and used by UserSpace CNI to interface with the
VPP GO-API. The UserSpace CNI is a CNI implementation designed to implement
User Space networking (as apposed to kernel space networking), like DPDK based
applications. For example VPP and OVS-DPDK.


The UserSpace CNI, based on the input config data, uses this library to add
interfaces (memif and vhost-user interface) to a local VPP instance running on
the host and add that interface to a local network, like a L2 Bridge. The
UserSpace CNI then processes config data intended for the container and uses
this library to add that data to a Database the container can consume.

# How to Build
## Install VPP
The VPP CNI opens a GO Channel to the the local VPP instance based on the VPP
API files. When VPP is installed, it copies it's json API files to
**/usr/share/vpp/api/**. VPP CNI uses these files to compile against and generate
the properly version messages to the local VPP Instance. So to build the VPP
CNI, VPP must be installed (or the proper json files must be in
**/usr/share/vpp/api/**).


There are several ways to install VPP. This code is based on a fixed release
VPP (VPP 18.04 initially), so it is best to install a released version (even
though it is possible to build your own).

### Prerequisites
Below are the VPP prerequisites:
* ***Hugepages:*** VPP requires 2M Hugepages. By default, VPP uses 1024
hugepages. If hugepagges are not configured, on install VPP will allocate
them. This is primarily an issue if you are running in a VM that does not
already have hugepage backing, especially when you reboot the VM. If you
would like to change the number of hugepages VPP uses, after installing VPP,
edit **/etc/sysctl.d/80-vpp.conf**. However, once VPP has been installed, the
default value has been applied. To reduce the number of hugepages, use:
   vm.nr_hugepages=512  
   vm.max_map_count=2048  
   kernel.shmmax=1073741824  
* ***SELinux:*** VPP works with SELinux enabled, but when running with
containers, work still needs to be done. Set SELinux to permissive.

### Install
To install VPP on CentOS from NFV SIG:
```
sudo yum install centos-release-fdio
sudo yum install vpp*
```

OR - To install from the VPP Nexus Repo:
```
vi /etc/yum.repos.d/fdio-stable-1804.repo
[fdio-stable-1804]
name=fd.io stable/1804 branch latest merge
baseurl=https://nexus.fd.io/content/repositories/fd.io.stable.1804.centos7/
enabled=1
gpgcheck=0

sudo yum install vpp*
```

To start and enable VPP:
```
sudo systemctl start vpp
sudo systemctl enable vpp
```

For installing VPP on other distros, see:
https://wiki.fd.io/view/VPP/Installing_VPP_binaries_from_packages

# Test
UserSpace CNI README.md describes how to test with VPP CNI. Below are a
few notes regarding packages in this repo.

## vpp-app
The ***vpp-app*** is intended to run in a container. It leverages the VPP CNI code
to consume interfaces in the container.

## cnivpp/docker/vpp-centos-userspace-cni/
The docker image ***vpp-centos-userspace-cni*** runs a VPP instance and the
vpp-app at startup. 

## cnivpp/vppdb
vppdb is use to store data in a DB. For the local VPP instance, the vppdb is
used to store the swIndex generated when the interface is created. It is used
later to delete the interface. The vppdb is also used to pass configuration
data to the container so the container can consume the interface.


The initial implementation of the DB is just json data written to a file.
This can be expanded at a later date to write to something like an etcd DB.


The following filenames and directory structure is used to store the data:
* ***Host***:
  * ***/var/run/vpp/cni/data/***:
    * ***local-<ContainerId:12>-<ifname>.json***: Used to store local data
needed to delete and cleanup.

  * ***/var/run/vpp/cni/shared/***: Not a database directory, but this directory
is used for interface socket files, for example: ***memif-<ContainerId:12>-<ifname>.sock***
This directory is mapped into that container as the same directory in the container.

  * ***/var/run/vpp/cni/<ContainerId>/***: This directory is mapped into that container
as ***/var/run/vpp/cni/data/***, so appears to the container as its local data
directory. This is where the container writes its
***local-<ContainerId:12>-<ifname>.json*** file described above.
    * ***remote-<ContainerId:12>-<ifname>.json***: This file contains the configuration
to apply the interface in the container. The data is the same json data passed into
the UserSpace CNI (define in **user-space-net-plugin/usrsptypes/usrsptypes.go**), but
the Container data has been copied into the Host data label. The vpp-app processes the
data as local data. Once this data is read in the container, the vpp-app deletes the
file.
    * ***addData-<ContainerId:12>-<ifname>.json***: This file is used to pass
additional data into the the container, which is not defined by **usrsptypes.go**.
This includes the ContianerId itself, and the results from the IPAM plugin that
were processed locally. Once this data is read in the container, the vpp-app deletes
the file.

* ***Container***:
  * ***/var/run/vpp/cni/data/***: Mapped from ***/var/run/vpp/cni/<ContainerId>/***
on the host.
  * ***/var/run/vpp/cni/shared/***: Mapped from ***/var/run/vpp/cni/shared/*** on
the host.

