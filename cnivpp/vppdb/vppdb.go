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

package vppdb


import (
	"fmt"
	"encoding/json"
	"io/ioutil"
	"io"
	"os"
	"path/filepath"

	"github.com/Billy99/user-space-net-plugin/usrsptypes"
)

//
// Constants
//
const defaultBaseCNIDir = "/var/run/vpp/cni"
const defaultLocalCNIDir = "/var/run/vpp/cni/data"
const debugVppDb = false


//
// Types
//

// This structure is a union of all the VPP data (for all types of
// interfaces) that need to be preserved for later use.
type VppSavedData struct {
        SwIfIndex      uint32 `json:"swIfIndex"`       // Software Index, used to access the created interface, needed to delete interface.
        MemifSocketId  uint32 `json:"memifSocketId"`   // Memif SocketId, used to access the created memif Socket File, used for debug only.
}

// This structure is used to pass additional data outside of the usrsptypes date into the container.
type additionalData struct {
        ContainerId    string                  `json:"containerId"`  // ContainerId used locally. Used in several place, namely in the socket filenames.
	IPData         usrsptypes.IPDataType   `json:"ipData"`       // Data structure returned from IPAM plugin. 
}


//
// API Functions
//

// saveVppConfig() - Some data needs to be saved, like the swIfIndex, for cmdDel().
//  This function squirrels the data away to be retrieved later.
func SaveVppConfig(conf *usrsptypes.NetConf, containerID string, data *VppSavedData) error {

	// Current implementation is to write data to a file with the name:
	//   /var/run/vpp/cni/data/local-<ContainerId:12>-<If0name>.json
	//   OLD: /var/run/vpp/cni/<ContainerId>/local-<If0name>.json

        fileName := fmt.Sprintf("local-%s-%s.json", containerID[:12], conf.If0name)
        if dataBytes, err := json.Marshal(data); err == nil {
                sockDir := defaultLocalCNIDir

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

		if debugVppDb {
                	fmt.Printf("SAVE FILE: swIfIndex=%d path=%s dataBytes=%s\n", data.SwIfIndex, path, dataBytes)
		}
                return ioutil.WriteFile(path, dataBytes, 0644)
        } else {
                return fmt.Errorf("ERROR: serializing delegate VPP saved data: %v", err)
        }
}

func LoadVppConfig(conf *usrsptypes.NetConf, containerID string, data *VppSavedData) (error) {

	fileName := fmt.Sprintf("local-%s-%s.json", containerID[:12], conf.If0name)
	sockDir := defaultLocalCNIDir
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
	FileCleanup(sockDir, path)

        return nil
}


//
// Functions for processing Remote Configs (configs for within a Container)
//

// saveRemoteConfig() - When a config read on the host is for a Container,
//      flip the location and write the data to a file. When the Container
//      comes up, it will read the file via () and delete the file. This function
//      writes the file.
func SaveRemoteConfig(conf *usrsptypes.NetConf, ipData usrsptypes.IPDataType, containerID string) error {

	var dataCopy usrsptypes.NetConf
	var addData additionalData

	// Current implementation is to write data to a file with the name:
	//   /var/run/vpp/cni/<ContainerId>/remote-<If0name>.json
	//   /var/run/vpp/cni/<ContainerId>/addData-<If0name>.json

	sockDir  := filepath.Join(defaultBaseCNIDir, containerID)

	if _, err := os.Stat(sockDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(sockDir, 0700); err != nil {
				return err
			}
		} else {
			return err
		}
	}


	//
	// Convert the remote configuration into a local configuration
	//
	dataCopy = *conf
	dataCopy.HostConf = dataCopy.ContainerConf
	dataCopy.ContainerConf = usrsptypes.UserSpaceConf{}

	// IPAM is processed by the host and sent to the Container. So blank out what was already processed.
	dataCopy.IPAM.Type = ""

	// Convert empty variables to valid data based on the original HostConf
	if dataCopy.HostConf.Engine == "" {
		dataCopy.HostConf.Engine = conf.HostConf.Engine
	}
	if dataCopy.HostConf.IfType == "" {
		dataCopy.HostConf.IfType = conf.HostConf.IfType
	}
	if dataCopy.HostConf.NetType == "" {
		dataCopy.HostConf.NetType = "interface"
	}

	if dataCopy.HostConf.IfType == "memif" {
		if dataCopy.HostConf.MemifConf.Role == "" {
			if conf.HostConf.MemifConf.Role == "master" {
				dataCopy.HostConf.MemifConf.Role = "slave"
			} else {
				dataCopy.HostConf.MemifConf.Role = "master"
			}	
		}
		if dataCopy.HostConf.MemifConf.Mode == "" {
			dataCopy.HostConf.MemifConf.Mode =  conf.HostConf.MemifConf.Mode
		}
	} else if dataCopy.HostConf.IfType == "vhostuser" {
		if dataCopy.HostConf.VhostConf.Mode == "" {
			if conf.HostConf.VhostConf.Mode == "client" {
				dataCopy.HostConf.VhostConf.Mode = "server"
			} else {
				dataCopy.HostConf.VhostConf.Mode = "client"
			}	
		}
	}


	//
	// Gather the additional data
	//
	addData.ContainerId = containerID
	addData.IPData      = ipData


	//
	// Marshall data and write to file
	//
	fileName := fmt.Sprintf("remote-%s.json", dataCopy.If0name)
	path     := filepath.Join(sockDir, fileName)

        dataBytes, err := json.Marshal(dataCopy)
	
	if err == nil {
		if debugVppDb {
			fmt.Printf("SAVE FILE: path=%s dataBytes=%s", path, dataBytes)
		}
		err = ioutil.WriteFile(path, dataBytes, 0644)
	} else {
		return fmt.Errorf("ERROR: serializing REMOTE NetConf data: %v", err)
	}


	if err == nil {
		fileName = fmt.Sprintf("addData-%s.json", dataCopy.If0name)
		path     = filepath.Join(sockDir, fileName)

		dataBytes, err = json.Marshal(addData)
        
		if err == nil {
			if debugVppDb {
		        	fmt.Printf("SAVE FILE: path=%s dataBytes=%s", path, dataBytes)
			}
			err = ioutil.WriteFile(path, dataBytes, 0644)
		} else {
			return fmt.Errorf("ERROR: serializing ADDDATA NetConf data: %v", err)
		}
	}

	return err
}


func FindRemoteConfig() (bool, usrsptypes.NetConf, usrsptypes.IPDataType, string, error) {
	var conf usrsptypes.NetConf
	var addData additionalData

	//
	// Find Primary input file
	//
	found, dataBytes, err := findFile(filepath.Join(defaultLocalCNIDir, "remote-*.json"))

	if err == nil {
		if found {
			if err = json.Unmarshal(dataBytes, &conf); err != nil {
				return found, conf, addData.IPData, addData.ContainerId, fmt.Errorf("failed to parse Remote config: %v", err)
			}


			//
			// Since Primary input was found, look for Additional Data file.
			//
			found, dataBytes, err = findFile(filepath.Join(defaultLocalCNIDir, "addData-*.json"))
			if err == nil {
				if found {
					if err = json.Unmarshal(dataBytes, &addData); err != nil {
						return found, conf, addData.IPData, addData.ContainerId, fmt.Errorf("failed to parse AddData config: %v", err)
					}
 				} else {
					return found, conf, addData.IPData, addData.ContainerId, fmt.Errorf("failed to read AddData config: %v", err)
				}
			}
 		} else {
			return found, conf, addData.IPData, addData.ContainerId, fmt.Errorf("failed to read Remote config: %v", err)
		}
	}
	
	return found, conf, addData.IPData, addData.ContainerId, err
}

// CleanupRemoteConfig() - When a config read on the host is for a Container,
//      the data to a file. This function cleans up the remaining files.
func CleanupRemoteConfig(conf *usrsptypes.NetConf, containerID string) {

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
func FileCleanup(directory string, filepath string) (err error) {

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


func findFile(filePath string) (bool, []byte, error) {
	var found bool = false

	if debugVppDb {
		fmt.Println(filePath)
	}
	matches, err := filepath.Glob(filePath)

	if err != nil {
		if debugVppDb {
			fmt.Println(err)
		}
		return found, nil, err
	}

	if debugVppDb {
		fmt.Println(matches)
	}

	for i := range matches {
		if debugVppDb {
                	fmt.Printf("PROCESSING FILE: path=%s\n", matches[i])
		}

		found = true

		if dataBytes, err := ioutil.ReadFile(matches[i]); err == nil {
			if debugVppDb {
	                	fmt.Printf("FILE DATA:\n%s\n", dataBytes)
			}

			// Delete file (and directory if empty)
			FileCleanup("", matches[i])

			return found, dataBytes, err
 		} else {
			return found, nil, fmt.Errorf("failed to read Remote config: %v", err)
		}
	}
	
	return found, nil, err
}
