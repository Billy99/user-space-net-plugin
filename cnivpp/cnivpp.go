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

	"github.com/Billy99/user-space-net-plugin/usrsptypes"

	"github.com/Billy99/cnivpp/api/infra"
	"github.com/Billy99/cnivpp/api/memif"
	"github.com/Billy99/cnivpp/api/bridge"

	"github.com/Billy99/cnivpp/vppdb"
)

//
// Constants
//
const (
	dbgBridge = true
	dbgMemif = true
)


//
// Types
//



//
// API Functions
//
func CniVppAdd(conf *usrsptypes.NetConf, containerID string) error {
	var err error
	var data vppdb.VppSavedData

	if conf.UserSpaceConf.Location == "local" {
		if conf.UserSpaceConf.Type == "memif" {
			err = addLocalDeviceMemif(conf, &data)
		} else if conf.UserSpaceConf.Type == "vhostuser" {
			return errors.New("GOOD: Found UserSpace Type:" + conf.UserSpaceConf.Type)
		} else {
			return errors.New("ERROR: Unknown UserSpace Type:" + conf.UserSpaceConf.Type)
		}

		if err != nil {
			return err
		}

		err = vppdb.SaveVppConfig(conf, containerID, &data)

		if err != nil {
			return err
		}
	} else if conf.UserSpaceConf.Location == "remote" {
		return vppdb.SaveRemoteConfig(conf, containerID)
	} else {
		return errors.New("ERROR: Unknown Location Type:" + conf.UserSpaceConf.Location)
	}

	return err
}

func CniVppDel(conf *usrsptypes.NetConf, containerID string) error {
	var data vppdb.VppSavedData

	// Retrived squirreled away data needed for processing delete
	err := vppdb.LoadVppConfig(conf, containerID, &data)

	if err != nil {
		return err
	}


	if conf.UserSpaceConf.Location == "local" {
		if conf.UserSpaceConf.Type == "memif" {
			return delLocalDeviceMemif(conf, &data)
		} else if conf.UserSpaceConf.Type == "vhostuser" {
			return errors.New("GOOD: Found UserSpace Type:" + conf.UserSpaceConf.Type)
		} else {
			return errors.New("ERROR: Unknown UserSpace Type:" + conf.UserSpaceConf.Type)
		}
	} else if conf.UserSpaceConf.Location == "remote" {
		vppdb.CleanupRemoteConfig(conf,containerID)
	} else {
		return errors.New("ERROR: Unknown Location Type:" + conf.UserSpaceConf.Location)
	}

	return err
}


func CniContainerConfig() (bool, error) {

	found, conf, err := vppdb.FindRemoteConfig()

	if err == nil {
		if found {
			err = CniVppAdd(&conf, "")

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
func addLocalDeviceMemif(conf *usrsptypes.NetConf, data *vppdb.VppSavedData) (err error) {
	var vppCh vppinfra.ConnectionData


	// Validate and convert input data
	var bridgeDomain uint32 = uint32(conf.UserSpaceConf.BridgeConf.BridgeId)
	var memifSocketId uint32 = uint32(conf.UserSpaceConf.MemifConf.SocketId)
	var memifSocketFile string = conf.UserSpaceConf.MemifConf.SocketFile
	var memifRole vppmemif.MemifRole
	var memifMode vppmemif.MemifMode

	if conf.UserSpaceConf.MemifConf.Role == "master" {
		memifRole = vppmemif.RoleMaster
	} else if conf.UserSpaceConf.MemifConf.Role == "slave" {
		memifRole = vppmemif.RoleSlave
	} else {
		return errors.New("ERROR: Invalid MEMIF Role:" + conf.UserSpaceConf.MemifConf.Role)
	}

	if conf.UserSpaceConf.MemifConf.Mode == "ethernet" {
		memifMode = vppmemif.ModeEthernet
	} else if conf.UserSpaceConf.MemifConf.Mode == "ip" {
		memifMode = vppmemif.ModeIP
	} else if conf.UserSpaceConf.MemifConf.Mode == "inject-punt" {
		memifMode = vppmemif.ModePuntInject
	} else {
		return errors.New("ERROR: Invalid MEMIF Mode:" + conf.UserSpaceConf.MemifConf.Mode)
	}

	// Set log level
	//   Logrus has six logging levels: DebugLevel, InfoLevel, WarningLevel, ErrorLevel, FatalLevel and PanicLevel.
	//core.SetLogger(&logrus.Logger{Level: logrus.InfoLevel})


	// Create Channel to pass requests to VPP
	vppCh,err = vppinfra.VppOpenCh()
	if err != nil {
		return
	}
	defer vppinfra.VppCloseCh(vppCh)


	// Compatibility Checks
	err = vppbridge.BridgeCompatibilityCheck(vppCh.Ch)
	if err != nil {
		return
	}
	err = vppmemif.MemifCompatibilityCheck(vppCh.Ch)
	if err != nil {
		return
	}


	// Create Memif Socket
	err = vppmemif.CreateMemifSocket(vppCh.Ch, memifSocketId, memifSocketFile)
	if err != nil {
		fmt.Println("Error:", err)
		return
	} else {
		fmt.Println("MEMIF SOCKET", memifSocketId, memifSocketFile, "created")
		if dbgMemif {
			vppmemif.DumpMemifSocket(vppCh.Ch)
		}
	}


	// Create MemIf Interface
	data.SwIfIndex, err = vppmemif.CreateMemifInterface(vppCh.Ch, memifSocketId, memifRole, memifMode)
	if err != nil {
		fmt.Println("Error:", err)
		return
	} else {
		fmt.Println("MEMIF", data.SwIfIndex, "created", conf.UserSpaceConf.Ifname)
		if dbgMemif {
			vppmemif.DumpMemif(vppCh.Ch)
		}
	}


	// Create Network
	if conf.UserSpaceConf.NetType == "bridge" {

		// Add MemIf to Bridge. If Bridge does not exist, AddBridgeInterface()
		// will create.
		err = vppbridge.AddBridgeInterface(vppCh.Ch, bridgeDomain, data.SwIfIndex)
		if err != nil {
			fmt.Println("Error:", err)
			return
		} else {
			fmt.Printf("INTERFACE %d add to BRIDGE %d\n", data.SwIfIndex, bridgeDomain)
			if dbgBridge {
				vppbridge.DumpBridge(vppCh.Ch, bridgeDomain)
			}
		}
	}

	return
}

func delLocalDeviceMemif(conf *usrsptypes.NetConf, data *vppdb.VppSavedData) (err error) {

	var vppCh vppinfra.ConnectionData

	// Validate and convert input data
	var bridgeDomain uint32 = uint32(conf.UserSpaceConf.BridgeConf.BridgeId)

	fmt.Printf("INTERFACE %d retrieved from CONF - attempt to DELETE Bridge %d\n", data.SwIfIndex, bridgeDomain)

	// Create Channel to pass requests to VPP
	vppCh,err = vppinfra.VppOpenCh()
	if err != nil {
		return
	}
	defer vppinfra.VppCloseCh(vppCh)


	// Compatibility Checks
	err = vppbridge.BridgeCompatibilityCheck(vppCh.Ch)
	if err != nil {
		return
	}
	err = vppmemif.MemifCompatibilityCheck(vppCh.Ch)
	if err != nil {
		return
	}


	// Remove MemIf from Bridge. RemoveBridgeInterface() will delete Bridge if
	// no more interfaces are associated with the Bridge.
	err = vppbridge.RemoveBridgeInterface(vppCh.Ch, bridgeDomain, data.SwIfIndex)

	if err != nil {
		fmt.Println("Error:", err)
		return
	} else {
		fmt.Printf("INTERFACE %d removed from BRIDGE %d\n", data.SwIfIndex, bridgeDomain)
		if dbgBridge {
			vppbridge.DumpBridge(vppCh.Ch, bridgeDomain)
		}
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

	return
}

