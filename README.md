# User Space CNI Plugin
This plugin is based on the **Vhostuser CNI plugin** provided by Intel
(https://github.com/intel/vhost-user-net-plugin). This rewrite was done to
address the following deficienies:
* **Vhostuser CNI plugin** is limited to vhost-user. Want to be able to use
other types of implementations.
* **Vhostuser CNI plugin** is written in GO. It is currently calling a python
script passed in from the input json. The python script then builds up a CLI
command (either VPP or OVS) and then executes the command in a shell command.
VPP has a GO API and would like to take advantage of that.

This code is a work in progress and has it's own set of deficienies:
* The input structures define in *usrsptypes* may not be the typical CNI layout
and probably need some adjustments.
* There is spot in the code to branch to OVS or Linux or some other
implementation, but only VPP has been implemented.
* Currently, all implentations are compiled in. Not a way to currently only
link in the implementation that are desired.
* Have only tested with the scripts provided with the Container Network
Interface (CNI) project. Have not tested with Multus or Kubernetes.
* Moved from a build script to a simple make file. Log term probably need
to go back to the build script, or at least add *install* functionality.
Only had one file to compile so went with simplicity for now. Make/Build
are not my strong suit.


# Build
## CNI VPP Library
This package currently depends on the **cnivpp** library, which is still being
developed and is a moving target. So it has not been added to the *vendor*
directory yet. All other dependancies are in the *vendor* directory (copied
from **Vhostuser CNI plugin**, so at the same version). To get the **cnivpp**
library:
```
   cd $GOPATH/src/
   go get github.com/Billy99/cnivpp
```

## User Space CNI Plugin
To get and build the **userspace** plugin:
```
   cd $GOPATH/src/
   go get github.com/Billy99/user-space-net-plugin
   cd github.com/Billy99/user-space-net-plugin
   make
```

Once the binary is built, it needs to be copied to the CNI directory:
```
   cp userspace/userspace $CNI_PATH/.
```

To perform a make clean:
```
   make clean
```


# Test

**TBD** - Haven't run this in a clean system. May need a few tweaks.

There are a few environmental variables used in this test. Here is an example:
```
   cat ~/.bashrc
   :
   export GOPATH=~/go
   export CNI_PATH=$GOPATH/src/github.com/containernetworking/plugins/bin

```

In order to test, a container with VPP 18.04 and vpp-app has been created:
```
  docker pull bmcfall/vpp-centos-userspace-cni
```

Setup your configuration files in your CNI directory. An example is
*/etc/cni/net.d/*.

**NOTE:** The *userspace* nectconf definition is still a work in progress. So
the example below is just an example, see *usrsptypes* for latest definitions.

Setup local configuration (on host):
```
sudo vi /etc/cni/net.d/90-userspace.conf 
{
        "type": "userspace",
        "name": "memif-network",
        "if0name": "net0",
        "userspace": {
                "type": "memif",
                "owner": "vpp",
                "location": "local",
                "netType": "bridge",
                "memif": {
                        "role": "master",
                        "mode": "ethernet",
                        "socketId": 3,
                        "socketFile": "/var/run/vpp/cni/shared/memif-3.sock"
                },
                "bridge": {
                        "bridgeId": 4
                }
        }
}
```

Setup local configuration (on host):
```
sudo vi /etc/cni/net.d/90-userspace.conf 
{
        "type": "userspace",
        "name": "memif-network",
        "if0name": "net0",
        "userspace": {
                "type": "memif",
                "owner": "vpp",
                "location": "remote",
                "netType": "bridge",
                "memif": {
                        "role": "slave",
                        "mode": "ethernet",
                        "socketId": 3,
                        "socketFile": "/var/run/vpp/cni/shared/memif-3.sock"
                },
                "bridge": {
                        "bridgeId": 4
                }
        }
}
```

To test, currently using a local script (copied from CNI scripts:
https://github.com/containernetworking/cni/blob/master/scripts/docker-run.sh).
To run script:
```
   cd $GOPATH/src/github.com/containernetworking/cni/scripts
   sudo CNI_PATH=$CNI_PATH GOPATH=$GOPATH ./scripts/vpp-docker-run.sh -it --privileged vpp-centos-userspace-cni
```

**NOTE:** The *vpp-docker-run.sh* script mounts some volumes in the container. Change as needed:
* *-v /var/run/vpp/cni/shared:/var/run/vpp/cni/shared:rw*
  * Default location in VPP to create sockets is */var/run/vpp/*. Socket files (memif or vhost-user)
are passed to the container through a subdirectory of this base directory..
* *-v /var/run/vpp/cni/$contid:/var/run/vpp/cni/data:rw*
  * Current implementation is to write the remote configuration into a file and share the directory
with the conatiner, which is the volume mapping. Directory is currently hard coded.
* *--device=/dev/hugepages:/dev/hugepages*
  * VPP requires hugepages, so need to map hugepoages into conatiner.

In the container, you should see the vpp-app ouput the message sequence of
its communication with local VPP (VPP in the container) and some database
dumps interleaved.

To verify the local config, in another window:
```
vppctl show interface
vppclt show mode
vppctl show memif
```

## Debug
The *vpp-centos-userspace-cni* container runs a script at startup (in Dockefile CMD command) which
starts VPP and then runs *vpp-app*. Assuming the same notes above, to see what is happening in the container,
cause *vpp-centos-userspace-cni* container to start in bash and skip the script, then run VPP and *vpp-app* manually: 
```
   cd $GOPATH/src/github.com/containernetworking/cni/scripts
   sudo CNI_PATH=$CNI_PATH GOPATH=$GOPATH ./scripts/vpp-docker-run.sh -it --privileged vpp-centos-userspace-cni bash
   
   /* Within Container: */
   vpp -c /etc/vpp/startup.conf &
   vpp-app
```

