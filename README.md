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

**TBD** - Coming soon ...

