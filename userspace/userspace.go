// Copyright 2017 Intel Corp.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"

	"github.com/containernetworking/cni/pkg/ipam"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"

	"github.com/Billy99/cnivpp/cnivpp"

	"github.com/Billy99/user-space-net-plugin/usrsptypes"
)


func init() {
	// this ensures that main runs only on main thread (thread group leader).
	// since namespace ops (unshare, setns) are done for a single thread, we
	// must ensure that the goroutine does not jump from OS thread to thread
	runtime.LockOSThread()
}

//
// Local functions
//

// loadNetConf() - Unmarshall the inputdata into the NetConf Structure 
func loadNetConf(bytes []byte) (*usrsptypes.NetConf, error) {
	n := &usrsptypes.NetConf{}
	if err := json.Unmarshal(bytes, n); err != nil {
		return nil, fmt.Errorf("failed to load netconf: %v", err)
	}

	return n, nil
}

const NET_CONFIG_TEMPLATE = `{
	"ipAddr": "%s/32",
	"macAddr": "%s",
	"gateway": "169.254.1.1",
	"gwMac": "%s"
}
`

func GenerateRandomMacAddress() string {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}

	// Set the local bit and make sure not MC address
	macAddr := fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		(buf[0]|0x2)&0xfe, buf[1], buf[2],
		buf[3], buf[4], buf[5])
	return macAddr
}

// SetupContainerNetwork Write the configuration to file
func SetupContainerNetwork(conf *usrsptypes.NetConf, containerID, containerIP string) {
	//args := []string{"config", conf.VhostConf.Vhostname, containerIP, conf.VhostConf.IfMac}
	//ExecCommand(conf.VhostConf.Vhosttool, args)

	// Write the configuration to file
	//config := fmt.Sprintf(NET_CONFIG_TEMPLATE, containerIP, conf.VhostConf.IfMac, conf.VhostConf.VhostMac)
	//fileName := fmt.Sprintf("%s-%s-ip4.conf", containerID[:12], conf.If0name)
	//sockDir := filepath.Join(conf.CNIDir, containerID)
	//configFile := filepath.Join(sockDir, fileName)
	//ioutil.WriteFile(configFile, []byte(config), 0644)
}

func cmdAdd(args *skel.CmdArgs) error {
	var result *types.Result
	var netConf *usrsptypes.NetConf

	// Convert the input bytestream into local NetConf structure
	netConf, err := loadNetConf(args.StdinData)
	if err != nil {
		return result.Print()
	}

	// Add the requested interface
	if netConf.UserSpaceConf.Owner == "vpp" {
		err = cnivpp.CniVppAdd(netConf, args.ContainerID)
		if err != nil {
			return err
		}
	} else if netConf.UserSpaceConf.Owner == "ovs-dpdk" {
		return errors.New("GOOD: Found UserSpace Owner:" + netConf.UserSpaceConf.Owner + " - NOT SUPPORTED")
	} else {
		return errors.New("ERROR: Unknown UserSpace Owner:" + netConf.UserSpaceConf.Owner)
	}

	if netConf.IPAM.Type != "" {
		// run the IPAM plugin and get back the config to apply
		result, err = ipam.ExecAdd(netConf.IPAM.Type, args.StdinData)
		if err != nil {
			return fmt.Errorf("failed to set up IPAM: %v", err)
		}
		if result.IP4 == nil {
			return errors.New("IPAM plugin returned missing IPv4 config")
		}

		containerIP := result.IP4.IP.IP.String()
		SetupContainerNetwork(netConf, args.ContainerID, containerIP)
	}

	return result.Print()
}

func cmdDel(args *skel.CmdArgs) error {
	var result *types.Result
	var netConf *usrsptypes.NetConf

	// Convert the input bytestream into local NetConf structure
	netConf, err := loadNetConf(args.StdinData)
	if err != nil {
		return result.Print()
	}

	// Delete the requested interface
	if netConf.UserSpaceConf.Owner == "vpp" {
		err = cnivpp.CniVppDel(netConf, args.ContainerID)
		if err != nil {
			return err
		}
	} else if netConf.UserSpaceConf.Owner == "ovs-dpdk" {
		return errors.New("GOOD: Found UserSpace Owner:" + netConf.UserSpaceConf.Owner + " - NOT SUPPORTED")
	} else {
		return errors.New("ERROR: Unknown UserSpace Owner:" + netConf.UserSpaceConf.Owner)
	}

	if netConf.IPAM.Type != "" {
		return ipam.ExecDel(netConf.IPAM.Type, args.StdinData)
	}
	return nil
}

func main() {
	skel.PluginMain(cmdAdd, cmdDel)
}
