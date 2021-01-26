// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2017-2018 Canonical Ltd
// Copyright (C) 2018-2021 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"context"
	"fmt"
	"time"

	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v2/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v2/di"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/errors"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/v2/dtos"
	commonDTO "github.com/edgexfoundry/go-mod-core-contracts/v2/v2/dtos/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/v2/dtos/requests"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/v2/models"
	"github.com/google/uuid"

	"github.com/edgexfoundry/device-sdk-go/v2/internal/common"
	"github.com/edgexfoundry/device-sdk-go/v2/internal/container"
	"github.com/edgexfoundry/device-sdk-go/v2/internal/v2/cache"
)

func LoadDevices(deviceList []common.DeviceConfig, dic *di.Container) errors.EdgeX {
	lc := bootstrapContainer.LoggingClientFrom(dic.Get)
	serviceName := container.DeviceServiceFrom(dic.Get).Name
	var addDevicesReq = make([]requests.AddDeviceRequest, len(deviceList))

	lc.Debug("loading pre-define devices from configuration")
	for _, d := range deviceList {
		if _, ok := cache.Devices().ForName(d.Name); ok {
			lc.Debugf("device %s exists, using the existing one", d.Name)
			continue
		} else {
			lc.Debugf("device %s doesn't exist, creating a new one", d.Name)
			deviceDTO, err := createDeviceDTO(serviceName, d)
			if err != nil {
				return err
			}
			req := requests.AddDeviceRequest{
				BaseRequest: commonDTO.BaseRequest{
					RequestId: uuid.New().String(),
				},
				Device:      deviceDTO,
			}
			addDevicesReq = append(addDevicesReq, req)
		}
	}

	dc := container.MetadataDeviceClientFrom(dic.Get)
	ctx := context.WithValue(context.Background(), common.CorrelationHeader, uuid.New().String())
	_, err := dc.Add(ctx, addDevicesReq)
	return err
}

func createDeviceDTO(name string, dc common.DeviceConfig) (deviceDTO dtos.Device,err errors.EdgeX) {
	prf, ok := cache.Profiles().ForName(dc.Profile)
	if !ok {
		errMsg := fmt.Sprintf("device profile %s for device %s doesn't exist", dc.Profile, dc.Name)
		return deviceDTO, errors.NewCommonEdgeX(errors.KindInvalidId, errMsg, nil)
	}

	device := models.Device{
		Name:           dc.Name,
		Description:    dc.Description,
		Protocols:      dc.Protocols,
		Labels:         dc.Labels,
		ProfileName:    prf.Name,
		ServiceName:    name,
		AdminState:     models.Unlocked,
		AutoEvents:     dc.AutoEvents,
	}
	device.Created = time.Now().UnixNano() / int64(time.Millisecond)

	return dtos.FromDeviceModelToDTO(device), nil
}
