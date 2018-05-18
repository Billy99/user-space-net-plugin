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

package usrsptypes

import (
	"github.com/containernetworking/cni/pkg/types"
)

//
// Exported Types
//
type MemifConf struct {
	Role       string `json:"role"`       // Role of memif: master|slave
	Mode       string `json:"mode"`       // Mode of memif: ip|ethernet|inject-punt
	SocketId   int    `json:"socketId"`   // SocketId for memif
	SocketFile string `json:"socketFile"` // Socketfile for memif
}

type VhostConf struct {
	Mode       string `json:"mode"`       // vhost-user mode: client|server
	SocketFile string `json:"socketFile"` // Socketfile for vhost-user
}

type BridgeConf struct {
	BridgeId   int `json:"bridgeId"`         // Bridge Id 
	VlanId     int `json:"vlanId,onitempty"` // Optional VLAN Id
}

type UserSpaceConf struct {
	Type       string `json:"type"`              // Type of interface - memif|vhostuser|veth|tap
	Owner      string `json:"owner"`             // CNI Implementation - vpp|ovs|ovs-dpdk|host
	Location   string `json:"location"`          // Interface location - local|remote
	NetType    string `json:"netType"`           // Interface network type - none|bridge|ip|vxlan|mpls
	Ifname     string `json:"ifname,omitempty"`  // Interface name
	IfMac      string `json:"ifmac,omitempty"`   // Interface MAC Address
	MemifConf  MemifConf  `json:"memif,omitempty"`
	VhostConf  VhostConf  `json:"vhost,omitempty"`
	BridgeConf BridgeConf `json:"bridge,omitempty"`
}

type NetConf struct {
	types.NetConf
	UserSpaceConf  UserSpaceConf `json:"userspace,omitempty"`
	If0name        string        `json:"if0name"`
}

