// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2020-2021 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"github.com/edgexfoundry/go-mod-bootstrap/v2/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/v2/models"

	dsModels "github.com/edgexfoundry/device-sdk-go/v2/pkg/models"
)

// DeviceServiceName contains the name of device service struct in the DIC.
var DeviceServiceName = di.TypeInstanceToName(models.DeviceService{})

// ProtocolDiscoveryName contains the name of protocol discovery implementation in the DIC.
var ProtocolDiscoveryName = di.TypeInstanceToName((*dsModels.ProtocolDiscovery)(nil))

// ProtocolDriverName contains the name of protocol driver implementation in the DIC.
var ProtocolDriverName = di.TypeInstanceToName((*dsModels.ProtocolDriver)(nil))

// ManagerName contains the name of autoevent manager implementation in the DIC
var ManagerName = di.TypeInstanceToName((*dsModels.Manager)(nil))

// DeviceServiceFrom helper function queries the DIC and returns device service struct.
func DeviceServiceFrom(get di.Get) models.DeviceService {
	return get(DeviceServiceName).(models.DeviceService)
}

// ProtocolDiscoveryFrom helper function queries the DIC and returns protocol discovery implementation.
func ProtocolDiscoveryFrom(get di.Get) dsModels.ProtocolDiscovery {
	casted, ok := get(ProtocolDiscoveryName).(dsModels.ProtocolDiscovery)
	if ok {
		return casted
	}
	return nil
}

// ProtocolDriverFrom helper function queries the DIC and returns protocol driver implementation.
func ProtocolDriverFrom(get di.Get) dsModels.ProtocolDriver {
	return get(ProtocolDriverName).(dsModels.ProtocolDriver)
}

// ManagerFrom helper function queries the DIC and returns autoevent manager implementation
func ManagerFrom(get di.Get) dsModels.Manager {
	return get(ManagerName).(dsModels.Manager)
}
