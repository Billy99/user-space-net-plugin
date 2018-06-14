// Copyright (c) 2018 Red Hat.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//
// This module provides the library functions to implement the
// VPP UserSpace CNI implementation. The input to the library is json
// data defined in usrsptypes. If the configuration contains local data,
// the 'api' library is used to send the request to the local govpp-agent,
// which provisions the local VPP instance. If the configuration contains
// remote data, the database library is used to store the data, which is
// later read and processed locally by the remotes agent (vpp-app running
// in the container)
//

package cnivpp


import (
	"fmt"
	"errors"
	"os"
	"path/filepath"

	"github.com/Billy99/user-space-net-plugin/usrsptypes"

	"github.com/Billy99/cnivpp/api/bridge"
	"github.com/Billy99/cnivpp/api/infra"
	"github.com/Billy99/cnivpp/api/interface"
	"github.com/Billy99/cnivpp/api/memif"
	"github.com/Billy99/cnivpp/api/vhostuser"

	"github.com/Billy99/cnivpp/vppdb"
)

//
// Constants
//
const (
	dbgBridge = true
	dbgMemif = true
)

const defaultVPPSocketDir = "/var/run/vpp/cni/shared/"

//
// Types
//



//
// API Functions
//
func CniVppAddOnHost(conf *usrsptypes.NetConf, ipData usrsptypes.IPDataType, containerID string) error {
	var vppCh vppinfra.ConnectionData
	var err error
	var data vppdb.VppSavedData


	// Set log level
	//   Logrus has six logging levels: DebugLevel, InfoLevel, WarningLevel, ErrorLevel, FatalLevel and PanicLevel.
	//core.SetLogger(&logrus.Logger{Level: logrus.InfoLevel})


	// Create Channel to pass requests to VPP
	vppCh,err = vppinfra.VppOpenCh()
	if err != nil {
		return err
	}
	defer vppinfra.VppCloseCh(vppCh)


	// Make sure version of API structs used by CNI are same as used by local VPP Instance.
	err = compatibilityChecks(vppCh)
	if err != nil {
		return err
	}


	//
	// Create Local Interface
	//
	if conf.HostConf.IfType == "memif" {
		err = addLocalDeviceMemif(vppCh, conf, containerID, &data)
	} else if conf.HostConf.IfType == "vhostuser" {
		err = errors.New("GOOD: Found HostConf.IfType:" + conf.HostConf.IfType)
	} else {
		err = errors.New("ERROR: Unknown HostConf.IfType:" + conf.HostConf.IfType)
	}

	if err != nil {
		return err
	}


	//
        // Set interface to up (1)
	//
	err = vppinterface.SetState(vppCh.Ch, data.SwIfIndex, 1)
	if err != nil {
		fmt.Println("Error bringing memif interface UP:", err)
		return err
	}


	// Add L2 Network if supplied
	//
	if conf.HostConf.NetType == "bridge" {

		var bridgeDomain uint32 = uint32(conf.HostConf.BridgeConf.BridgeId)

		// Add Interface to Bridge. If Bridge does not exist, AddBridgeInterface()
		// will create.
		err = vppbridge.AddBridgeInterface(vppCh.Ch, bridgeDomain, data.SwIfIndex)
		if err != nil {
			fmt.Println("Error:", err)
			return err
		} else {
			fmt.Printf("INTERFACE %d added to BRIDGE %d\n", data.SwIfIndex, bridgeDomain)
			if dbgBridge {
				vppbridge.DumpBridge(vppCh.Ch, bridgeDomain)
			}
		}
	}


	//
	// Add L3 Network if supplied
	//
	if ipData.Address != "" {
		err = vppinterface.AddDelIpAddress(vppCh.Ch, data.SwIfIndex, 1, ipData)
		if err != nil {
			fmt.Println("Error:", err)
			return err
		} else {
			fmt.Printf("IP %s added to INTERFACE %d\n", data.SwIfIndex, ipData.Address)
		}
	}


	//
	// Save Create Data for Delete
	//
	err = vppdb.SaveVppConfig(conf, containerID, &data)

	if err != nil {
		return err
	}

	return err
}

func CniVppAddOnContainer(conf *usrsptypes.NetConf, ipData usrsptypes.IPDataType, containerID string) error {
	return vppdb.SaveRemoteConfig(conf, ipData, containerID)
}


func CniVppDelFromHost(conf *usrsptypes.NetConf, containerID string) error {
	var vppCh vppinfra.ConnectionData
	var data vppdb.VppSavedData
	var err error

	// Set log level
	//   Logrus has six logging levels: DebugLevel, InfoLevel, WarningLevel, ErrorLevel, FatalLevel and PanicLevel.
	//core.SetLogger(&logrus.Logger{Level: logrus.InfoLevel})


	// Create Channel to pass requests to VPP
	vppCh,err = vppinfra.VppOpenCh()
	if err != nil {
		return err
	}
	defer vppinfra.VppCloseCh(vppCh)


	// Retrieved squirreled away data needed for processing delete
	err = vppdb.LoadVppConfig(conf, containerID, &data)

	if err != nil {
		return err
	}


	//
	// Remove L2 Network if supplied
	//
	if conf.HostConf.NetType == "bridge" {

		// Validate and convert input data
		var bridgeDomain uint32 = uint32(conf.HostConf.BridgeConf.BridgeId)

		fmt.Printf("INTERFACE %d retrieved from CONF - attempt to DELETE Bridge %d\n", data.SwIfIndex, bridgeDomain)


		// Remove MemIf from Bridge. RemoveBridgeInterface() will delete Bridge if
		// no more interfaces are associated with the Bridge.
		err = vppbridge.RemoveBridgeInterface(vppCh.Ch, bridgeDomain, data.SwIfIndex)

		if err != nil {
			fmt.Println("Error:", err)
			return err
		} else {
			fmt.Printf("INTERFACE %d removed from BRIDGE %d\n", data.SwIfIndex, bridgeDomain)
			if dbgBridge {
				vppbridge.DumpBridge(vppCh.Ch, bridgeDomain)
			}
		}
	}


	//
	// Delete Local Interface
	//
	if conf.HostConf.IfType == "memif" {
		return delLocalDeviceMemif(vppCh, conf, containerID, &data)
	} else if conf.HostConf.IfType == "vhostuser" {
		return errors.New("GOOD: Found HostConf.Type:" + conf.HostConf.IfType)
	} else {
		return errors.New("ERROR: Unknown HostConf.Type:" + conf.HostConf.IfType)
	}

	return err
}

func CniVppDelFromContainer(conf *usrsptypes.NetConf, containerID string) error {
	vppdb.CleanupRemoteConfig(conf,containerID)
	return nil
}


func CniContainerConfig() (bool, error) {

	found, conf, ipData, containerId, err := vppdb.FindRemoteConfig()

	if err == nil {
		if found {
			fmt.Println("ipData:")
			fmt.Println(ipData)

			err = CniVppAddOnHost(&conf, ipData, containerId)

			if err != nil {
				fmt.Println(err)
			}
		}
	}

	return found, err
}


//
// Local Functions
//

func compatibilityChecks(vppCh vppinfra.ConnectionData) (err error) {

	// Compatibility Checks
	err = vppmemif.MemifCompatibilityCheck(vppCh.Ch)
	if err != nil {
		return
	}

	err = vppbridge.BridgeCompatibilityCheck(vppCh.Ch)
	if err != nil {
		return
	}

	err = vppvhostuser.VhostUserCompatibilityCheck(vppCh.Ch)
	if err != nil {
		return
	}

	err = vppinterface.InterfaceCompatibilityCheck(vppCh.Ch)
	if err != nil {
		return
	}

	return
}

func addLocalDeviceMemif(vppCh vppinfra.ConnectionData, conf *usrsptypes.NetConf, containerID string, data *vppdb.VppSavedData) (err error) {
	var ok bool

	// Validate and convert input data
	var memifSocketFile string
	var memifRole vppmemif.MemifRole
	var memifMode vppmemif.MemifMode

	if memifSocketFile, ok = os.LookupEnv("USERSPACE_MEMIF_SOCKFILE"); ok == false {
		fileName := fmt.Sprintf("memif-%s-%s.sock", containerID[:12], conf.If0name)
		memifSocketFile = filepath.Join(defaultVPPSocketDir, fileName)
	}

	if conf.HostConf.MemifConf.Role == "master" {
		memifRole = vppmemif.RoleMaster
	} else if conf.HostConf.MemifConf.Role == "slave" {
		memifRole = vppmemif.RoleSlave
	} else {
		return errors.New("ERROR: Invalid MEMIF Role:" + conf.HostConf.MemifConf.Role)
	}

	if conf.HostConf.MemifConf.Mode == "" {
		conf.HostConf.MemifConf.Mode = "ethernet"
	}
	if conf.HostConf.MemifConf.Mode == "ethernet" {
		memifMode = vppmemif.ModeEthernet
	} else if conf.HostConf.MemifConf.Mode == "ip" {
		memifMode = vppmemif.ModeIP
	} else if conf.HostConf.MemifConf.Mode == "inject-punt" {
		memifMode = vppmemif.ModePuntInject
	} else {
		return errors.New("ERROR: Invalid MEMIF Mode:" + conf.HostConf.MemifConf.Mode)
	}


	// Create Memif Socket
	data.MemifSocketId, err = vppmemif.CreateMemifSocket(vppCh.Ch, memifSocketFile)
	if err != nil {
		fmt.Println("Error:", err)
		return
	} else {
		fmt.Println("MEMIF SOCKET", data.MemifSocketId, memifSocketFile, "created")
		if dbgMemif {
			vppmemif.DumpMemifSocket(vppCh.Ch)
		}
	}


	// Create MemIf Interface
	data.SwIfIndex, err = vppmemif.CreateMemifInterface(vppCh.Ch, data.MemifSocketId, memifRole, memifMode)
	if err != nil {
		fmt.Println("Error:", err)
		return
	} else {
		fmt.Println("MEMIF", data.SwIfIndex, "created", conf.If0name)
		if dbgMemif {
			vppmemif.DumpMemif(vppCh.Ch)
		}
	}

	return
}

func delLocalDeviceMemif(vppCh vppinfra.ConnectionData, conf *usrsptypes.NetConf, containerID string, data *vppdb.VppSavedData) (err error) {

	var ok bool
	var memifSocketFile string

	if memifSocketFile, ok = os.LookupEnv("USERSPACE_MEMIF_SOCKFILE"); ok == false {
		fileName := fmt.Sprintf("memif-%s-%s.sock", containerID[:12], conf.If0name)
		memifSocketFile = filepath.Join(defaultVPPSocketDir, fileName)
	}

	fmt.Println("Delete memif interface.")
	err = vppmemif.DeleteMemifInterface(vppCh.Ch, data.SwIfIndex)
	if err != nil {
		fmt.Println("Error:", err)
		return
	} else {
		fmt.Printf("INTERFACE %d deleted\n", data.SwIfIndex)
		if dbgMemif {
			vppmemif.DumpMemif(vppCh.Ch)
			vppmemif.DumpMemifSocket(vppCh.Ch)
		}
	}

	// Remove file
	err = vppdb.FileCleanup("", memifSocketFile)

	return
}

