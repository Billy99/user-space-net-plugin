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
	"encoding/json"
	"io/ioutil"
	"io"
	"os"
	"path/filepath"

	"github.com/intel/vhost-user-net-plugin/usrsptypes"

	"github.com/Billy99/vpp-agent/api/infra"
	"github.com/Billy99/vpp-agent/api/memif"
	"github.com/Billy99/vpp-agent/api/bridge"
)

//
// Constants
//
const (
	dbgBridge = true
	dbgMemif = true
)

const defaultBaseCNIDir = "/var/run/vpp/cni"
const defaultLocalCNIDir = "/var/run/vpp/cni/data"


//
// Types
//

// This structure is a union of all the VPP data (for all types of
// interfaces) that need to be preserved for later use.
type vppSavedData struct {
        SwIfIndex  uint32 `json:"swIfIndex"`     // Software Index, used to access the created interface
}


//
// API Functions
//
func CniVppAdd(conf *usrsptypes.NetConf, containerID string) error {
	var err error
	var data vppSavedData

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

		err = saveVppConfig(conf, containerID, &data)

		if err != nil {
			return err
		}
	} else if conf.UserSpaceConf.Location == "remote" {
		return saveRemoteConfig(conf, containerID)
	} else {
		return errors.New("ERROR: Unknown Location Type:" + conf.UserSpaceConf.Location)
	}

	return err
}

func CniVppDel(conf *usrsptypes.NetConf, containerID string) error {
	var data vppSavedData

	// Retrived squirreled away data needed for processing delete
	err := loadVppConfig(conf, containerID, &data)

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
		cleanupRemoteConfig(conf,containerID)
	} else {
		return errors.New("ERROR: Unknown Location Type:" + conf.UserSpaceConf.Location)
	}

	return err
}


func CniContainerConfig() (bool, error) {
	return findRemoteConfig()
}


//
// Local Functions
//
func addLocalDeviceMemif(conf *usrsptypes.NetConf, data *vppSavedData) (err error) {
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

func delLocalDeviceMemif(conf *usrsptypes.NetConf, data *vppSavedData) (err error) {

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

// saveVppConfig() - Some data needs to be saved, like the swIfIndex, or cmdDel().
//  This function squirrels the data away to be retrieved later.
func saveVppConfig(conf *usrsptypes.NetConf, containerID string, data *vppSavedData) error {

	// Current implementation is to write data to a file with the name:
	//   /var/run/vpp/cni/data/local-<If0name>.json
	//   OLD: /var/run/vpp/cni/<ContainerId>/local-<If0name>.json

        fileName := fmt.Sprintf("local-%s.json", conf.If0name)
        if dataBytes, err := json.Marshal(data); err == nil {
                sockDir := defaultLocalCNIDir
                // OLD: sockDir := filepath.Join(defaultCNIDir, containerID)

                if _, err := os.Stat(sockDir); err != nil {
                        if os.IsNotExist(err) {
                                if err := os.MkdirAll(sockDir, 0700); err != nil {
                                        return err
                                }
                        } else {
                                return err
                        }
                }

                path := filepath.Join(sockDir, fileName)

                fmt.Printf("SAVE FILE: swIfIndex=%d path=%s dataBytes=%s\n", data.SwIfIndex, path, dataBytes)
                return ioutil.WriteFile(path, dataBytes, 0644)
        } else {
                return fmt.Errorf("ERROR: serializing delegate VPP saved data: %v", err)
        }
}

func loadVppConfig(conf *usrsptypes.NetConf, containerID string, data *vppSavedData) (error) {

	fileName := fmt.Sprintf("local-%s.json", conf.If0name)
	sockDir := defaultLocalCNIDir
	// OLD: sockDir := filepath.Join(defaultCNIDir, containerID)
	path := filepath.Join(sockDir, fileName)

	if _, err := os.Stat(path); err == nil {
		if dataBytes, err := ioutil.ReadFile(path); err == nil {
			if err = json.Unmarshal(dataBytes, data); err != nil {
				return fmt.Errorf("ERROR: Failed to parse VPP saved data: %v", err)
			}
		} else {
			return fmt.Errorf("ERROR: Failed to read VPP saved data: %v", err)
		}

        } else {
		path = "";
	}

	// Delete file (and directory if empty)
	fileCleanup(sockDir, path)

        return nil
}


//
// Functions for processing Remote Configs (configs for within a Container)
//

// saveRemoteConfig() - When a config read on the host is for a Container,
//      flip the location and write the data to a file. When the Container
//      comes up, it will read the file via () and delete the file. This function
//      writes the file.
func saveRemoteConfig(conf *usrsptypes.NetConf, containerID string) error {

	// Current implementation is to write data to a file with the name:
	//   /var/run/vpp/cni/<ContainerId>/remote-<If0name>.json

	fileName := fmt.Sprintf("remote-%s.json", conf.If0name)
	sockDir  := filepath.Join(defaultBaseCNIDir, containerID)
	path     := filepath.Join(sockDir, fileName)

	if _, err := os.Stat(sockDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(sockDir, 0700); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	conf.UserSpaceConf.Location = "local"
        dataBytes, err := json.Marshal(conf)
	conf.UserSpaceConf.Location = "remote"
        
	if err == nil {
                fmt.Printf("SAVE FILE: path=%s dataBytes=%s", path, dataBytes)
                return ioutil.WriteFile(path, dataBytes, 0644)
        } else {
                return fmt.Errorf("ERROR: serializing REMOTE NetConf data: %v", err)
        }
}


func findRemoteConfig() (bool, error) {
	var found bool = false
	var conf usrsptypes.NetConf

	//sockDir  := filepath.Join(defaultCNIDir, containerID)
	//sockDir  := defaultCNIDir
	sockDir  := filepath.Join(defaultLocalCNIDir, "remote-*.json")

	fmt.Println(sockDir)
	matches, err := filepath.Glob(sockDir)

	if err != nil {
		fmt.Println(err)
		return found, err
	}

	fmt.Println(sockDir)
	fmt.Println(matches)

	for i := range matches {
                fmt.Printf("PROCESSING FILE: path=%s\n", matches[i])

		found = true

		if dataBytes, err := ioutil.ReadFile(matches[i]); err == nil {
			if err = json.Unmarshal(dataBytes, &conf); err != nil {
				return found, fmt.Errorf("failed to parse Remote config: %v", err)
			}

			// Delete file (and directory if empty)
			fileCleanup("", matches[i])


			// Process data
			err = CniVppAdd(&conf, "")

			if err != nil {
				fmt.Println(err)
				return found, err
			}
 		} else {
			return found, fmt.Errorf("failed to read Remote config: %v", err)
		}
	}
	
	return found,err
}

// cleanupRemoteConfig() - When a config read on the host is for a Container,
//      the data to a file. This function cleans up the remaining files.
func cleanupRemoteConfig(conf *usrsptypes.NetConf, containerID string) {

	// Current implementation is to write data to a file with the name:
	//   /var/run/vpp/cni/<ContainerId>/remote-<If0name>.json

	sockDir  := filepath.Join(defaultBaseCNIDir, containerID)

	if err := os.RemoveAll(sockDir); err != nil {
		fmt.Println(err)
	}
}


//
// Utility Functions
//

// This function deletes the input file (if provided) and the associated
// directory (if provided) if the directory is empty.
//  directory string - Directory file is located in, Use "" if directory
//    should remain unchanged.
//  filepath string - File (including directory) to be deleted. Use "" if
//    only the directory should be deleted.
func fileCleanup(directory string, filepath string) (err error) {

	// If File is provided, delete it.
	if filepath != "" {
		err = os.Remove(filepath)
		if err != nil {
			return fmt.Errorf("ERROR: Failed to delete file: %v", err)
		}
	}

	// If Directory is provided and it is empty, delete it.
	if directory != "" {
		f, dirErr := os.Open(directory)
		if dirErr == nil {
			 _, dirErr = f.Readdir(1)
			if dirErr == io.EOF {
				err = os.Remove(directory)
			}
		}
		f.Close()
	}

	return
}

