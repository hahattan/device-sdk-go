// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2018 Canonical Ltd
// Copyright (C) 2018-2020 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"

	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/v2/dtos"
	commonDTO "github.com/edgexfoundry/go-mod-core-contracts/v2/v2/dtos/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/v2/dtos/requests"
	contract "github.com/edgexfoundry/go-mod-core-contracts/v2/v2/models"
	"github.com/google/uuid"

	"github.com/edgexfoundry/device-sdk-go/v2/internal/common"
	"github.com/edgexfoundry/device-sdk-go/v2/internal/transformer"
	"github.com/edgexfoundry/device-sdk-go/v2/internal/v2/cache"
	dsModels "github.com/edgexfoundry/device-sdk-go/v2/pkg/models"
)

// processAsyncResults processes readings that are pushed from
// a DS implementation. Each is reading is optionally transformed
// before being pushed to Core Data.
// In this function, AsyncBufferSize is used to create a buffer for
// processing AsyncValues concurrently, so that events may arrive
// out-of-order in core-data / app service when AsyncBufferSize value
// is greater than or equal to two. Alternatively, we can process
// AsyncValues one by one in the same order by changing the AsyncBufferSize
// value to one.
func (s *DeviceService) processAsyncResults(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	defer func() {
		wg.Done()
	}()

	working := make(chan bool, s.config.Service.AsyncBufferSize)
	for {
		select {
		case <-ctx.Done():
			return
		case acv := <-s.asyncCh:
			go s.sendAsyncValues(acv, working)
		}
	}
}

// sendAsyncValues convert AsyncValues to event and send the event to CoreData
func (s *DeviceService) sendAsyncValues(acv *dsModels.AsyncValues, working chan bool) {
	working <- true
	defer func() {
		<-working
	}()
	readings := make([]contract.Reading, 0, len(acv.CommandValues))

	device, ok := cache.Devices().ForName(acv.DeviceName)
	if !ok {
		s.LoggingClient.Errorf("async - failed to find received device %s in cache", acv.DeviceName)
		return
	}

	for _, cv := range acv.CommandValues {
		// get the device resource associated with the rsp.RO
		dr, ok := cache.Profiles().DeviceResource(device.ProfileName, cv.DeviceResourceName)
		if !ok {
			s.LoggingClient.Errorf("async - failed to find device resource %s for device %s", cv.DeviceResourceName, acv.DeviceName)
			continue
		}

		// TODO: uncomment when transform package is ready for v2.
		//if s.config.Device.DataTransform {
		//	err := transformer.TransformReadResult(cv, dr.Properties.Value, s.LoggingClient)
		//	if err != nil {
		//		s.LoggingClient.Error(fmt.Sprintf("processAsyncResults - CommandValue (%s) transformed failed: %v", cv.String(), err))
		//
		//		if errors.As(err, &transformer.OverflowError{}) {
		//			cv = dsModels.NewStringValue(cv.DeviceResourceName, cv.Origin, transformer.Overflow)
		//		} else if errors.As(err, &transformer.NaNError{}) {
		//			cv = dsModels.NewStringValue(cv.DeviceResourceName, cv.Origin, transformer.NaN)
		//		} else {
		//			cv = dsModels.NewStringValue(cv.DeviceResourceName, cv.Origin, fmt.Sprintf("Transformation failed for device resource, with value: %s, property value: %v, and error: %v", cv.String(), dr.Properties.Value, err))
		//		}
		//	}
		//}

		//err := transformer.CheckAssertion(cv, dr.Properties.Value.Assertion, &device, s.LoggingClient, s.edgexClients.DeviceClient)
		//if err != nil {
		//	s.LoggingClient.Error(fmt.Sprintf("processAsyncResults - Assertion failed for device resource: %s, with value: %s and assertion: %s, %v", cv.DeviceResourceName, cv.String(), dr.Properties.Value.Assertion, err))
		//	cv = dsModels.NewStringValue(cv.DeviceResourceName, cv.Origin, fmt.Sprintf("Assertion failed for device resource, with value: %s and assertion: %s", cv.String(), dr.Properties.Value.Assertion))
		//}

		//ro, err := cache.Profiles().ResourceOperation(device.Profile.Name, cv.DeviceResourceName, common.GetCmdMethod)
		//if err != nil {
		//	s.LoggingClient.Debug(fmt.Sprintf("processAsyncResults - getting resource operation failed: %s", err.Error()))
		//} else if len(ro.Mappings) > 0 {
		//	newCV, ok := transformer.MapCommandValue(cv, ro.Mappings)
		//	if ok {
		//		cv = newCV
		//	} else {
		//		s.LoggingClient.Warn(fmt.Sprintf("processAsyncResults - Mapping failed for Device Resource Operation: %s, with value: %s, %v", ro.DeviceCommand, cv.String(), err))
		//	}
		//}

		event, err := application.comm
		reading := common.CommandValueToReading(cv, device.Name, dr.Properties.Value.MediaType, dr.Properties.Value.FloatEncoding)
		readings = append(readings, *reading)
	}

	// push to Core Data
	cevent := contract.Event{Device: device.Name, Readings: readings}
	event := &dsModels.Event{Event: cevent}
	event.Origin = common.GetUniqueOrigin()
	common.SendEvent(event, s.LoggingClient, s.edgexClients.EventClient)
}

// processAsyncFilterAndAdd filter and add devices discovered by
// device service protocol discovery.
func (s *DeviceService) processAsyncFilterAndAdd(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	defer func() {
		wg.Done()
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case devices := <-s.deviceCh:
			ctx := context.Background()
			pws := cache.ProvisionWatchers().All()
			for _, d := range devices {
				for _, pw := range pws {
					if whitelistPass(d, pw, s.LoggingClient) && blacklistPass(d, pw, s.LoggingClient) {
						if _, ok := cache.Devices().ForName(d.Name); ok {
							s.LoggingClient.Debugf("candidate discovered device %s already existed", d.Name)
							break
						}

						s.LoggingClient.Infof("adding discovered device %s to Edgex", d.Name)
						device := contract.Device{
							Name:        d.Name,
							Description: d.Description,
							Protocols:   d.Protocols,
							Labels:      d.Labels,
							ProfileName: pw.ProfileName,
							ServiceName: pw.ServiceName,
							AdminState:  pw.AdminState,
							AutoEvents:  pw.AutoEvents,
						}

						req := requests.AddDeviceRequest{
							BaseRequest: commonDTO.BaseRequest{
								RequestId: uuid.New().String(),
							},
							Device:      dtos.FromDeviceModelToDTO(device),
						}
						_, err := s.edgexClients.DeviceClient.Add(ctx, []requests.AddDeviceRequest{req})
						if err != nil {
							s.LoggingClient.Errorf("failed to create discovered device %s: %v", device.Name, err)
						} else {
							break
						}
					}
				}
			}
			s.LoggingClient.Debug("Filtered device addition finished")
		}
	}
}

func whitelistPass(d dsModels.DiscoveredDevice, pw contract.ProvisionWatcher, lc logger.LoggingClient) bool {
	// ignore the device protocol properties name
	for _, protocol := range d.Protocols {
		matchedCount := 0
		for name, regex := range pw.Identifiers {
			if value, ok := protocol[name]; ok {
				matched, err := regexp.MatchString(regex, value)
				if !matched || err != nil {
					lc.Debugf("Device %s's %s value %s did not match PW identifier: %s", d.Name, name, value, regex)
					break
				}
				matchedCount += 1
			}
		}
		// match succeed on all identifiers
		if matchedCount == len(pw.Identifiers) {
			return true
		}
	}
	return false
}

func blacklistPass(d dsModels.DiscoveredDevice, pw contract.ProvisionWatcher, lc logger.LoggingClient) bool {
	// a candidate should match none of the blocking identifiers
	for name, blacklist := range pw.BlockingIdentifiers {
		// ignore the device protocol properties name
		for _, protocol := range d.Protocols {
			if value, ok := protocol[name]; ok {
				for _, v := range blacklist {
					if value == v {
						lc.Debugf("Discovered Device %s's %s should not be %s", d.Name, name, value)
						return false
					}
				}
			}
		}
	}
	return true
}
