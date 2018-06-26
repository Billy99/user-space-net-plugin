// Copyright (c) 2017 Cisco and/or its affiliates.
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

// Binary simple-client is an example VPP management application that exercises the
// govpp API on real-world use-cases.
package vppinterface

// Generates Go bindings for all VPP APIs located in the json directory.
//go:generate binapi-generator --input-dir=../../bin_api --output-dir=../../bin_api

import (
	"fmt"
	"net"

	"github.com/Billy99/user-space-net-plugin/usrsptypes"

	"git.fd.io/govpp.git/api"
	"git.fd.io/govpp.git/core/bin_api/interfaces"
)


//
// Constants
//
const debugInterface = false


//
// API Functions
//
// Check whether generated API messages are compatible with the version
// of VPP which the library is connected to.
func InterfaceCompatibilityCheck(ch *api.Channel) (err error) {
	err = ch.CheckMessageCompatibility(
		&interfaces.SwInterfaceSetFlags{},
		&interfaces.SwInterfaceSetFlagsReply{},
		&interfaces.SwInterfaceAddDelAddress{},
		&interfaces.SwInterfaceAddDelAddressReply{},
	)
	if err != nil {
		if debugInterface {
			fmt.Println("VPP Interface failed compatibility")
		}
	}

	return err
}


// Attempt to set an interface state. isUp (1 = up, 0 = down)
func SetState(ch *api.Channel, swIfIndex uint32, isUp uint8) error {
	// Populate the Add Structure
	req := &interfaces.SwInterfaceSetFlags{
		SwIfIndex: swIfIndex,
		// 1 = up, 0 = down
		AdminUpDown: isUp, 
	}

	reply := &interfaces.SwInterfaceSetFlagsReply{}

	err := ch.SendRequest(req).ReceiveReply(reply)

	if err != nil {
		if debugInterface {
			fmt.Println("Error:", err)
		}
		return err
	}

	return nil
}


func AddDelIpAddress(ch *api.Channel, swIfIndex uint32, isAdd uint8, ipData usrsptypes.IPDataType) error {

	addr := net.ParseIP(ipData.Address)

	
	// Populate the Add Structure
	req := &interfaces.SwInterfaceAddDelAddress{
		SwIfIndex: swIfIndex,
		IsAdd: isAdd, // 1 = add, 0 = delete
		IsIpv6: ipData.IsIpv6,
		DelAll: 0,
		AddressLength: ipData.AddressLength,
		//Address: []byte(ipData.Address),
	}

	if ipData.IsIpv6 == 1 {
		req.Address = []byte(addr.To16())
	} else {
		req.Address = []byte(addr.To4())
	}

	if debugInterface {
		fmt.Println("IP Address")
		fmt.Println(req.Address)
	}

	reply := &interfaces.SwInterfaceAddDelAddressReply{}

	err := ch.SendRequest(req).ReceiveReply(reply)

	if err != nil {
		if debugInterface {
			fmt.Println("Error:", err)
		}
		return err
	}

	return nil
}

