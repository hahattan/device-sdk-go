// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2020 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"fmt"
	"net/http"

	"github.com/edgexfoundry/device-sdk-go/internal/autoevent"
	"github.com/edgexfoundry/device-sdk-go/internal/cache"
	sdkCommon "github.com/edgexfoundry/device-sdk-go/internal/common"
	"github.com/edgexfoundry/device-sdk-go/internal/container"
	"github.com/edgexfoundry/device-sdk-go/internal/telemetry"
	"github.com/edgexfoundry/go-mod-bootstrap/di"
	contractsV2 "github.com/edgexfoundry/go-mod-core-contracts/v2"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/dtos/common"
	"github.com/gorilla/mux"
)

// Ping handles the request to /ping endpoint. Is used to test if the service is working
// It returns a response as specified by the V2 API swagger in openapi/v2
func (c *RestController) Ping(writer http.ResponseWriter, request *http.Request, _ *di.Container) {
	response := common.NewPingResponse()
	c.sendResponse(writer, request, contractsV2.ApiPingRoute, response, http.StatusOK)
}

// Version handles the request to /version endpoint. Is used to request the service's versions
// It returns a response as specified by the V2 API swagger in openapi/v2
func (c *RestController) Version(writer http.ResponseWriter, request *http.Request, _ *di.Container) {
	response := common.NewVersionResponse(sdkCommon.ServiceVersion)
	c.sendResponse(writer, request, contractsV2.ApiVersionRoute, response, http.StatusOK)
}

// Config handles the request to /config endpoint. Is used to request the service's configuration
// It returns a response as specified by the V2 API swagger in openapi/v2
func (c *RestController) Config(writer http.ResponseWriter, request *http.Request, dic *di.Container) {
	response := common.NewConfigResponse(container.ConfigurationFrom(dic.Get))
	c.sendResponse(writer, request, contractsV2.ApiVersionRoute, response, http.StatusOK)
}

// Metrics handles the request to the /metrics endpoint, memory and cpu utilization stats
// It returns a response as specified by the V2 API swagger in openapi/v2
func (c *RestController) Metrics(writer http.ResponseWriter, request *http.Request, _ *di.Container) {
	telem := telemetry.NewSystemUsage()
	metrics := common.Metrics{
		MemAlloc:       telem.Memory.Alloc,
		MemFrees:       telem.Memory.Frees,
		MemLiveObjects: telem.Memory.LiveObjects,
		MemMallocs:     telem.Memory.Mallocs,
		MemSys:         telem.Memory.Sys,
		MemTotalAlloc:  telem.Memory.TotalAlloc,
		CpuBusyAvg:     uint8(telem.CpuBusyAvg),
	}

	response := common.NewMetricsResponse(metrics)
	c.sendResponse(writer, request, contractsV2.ApiMetricsRoute, response, http.StatusOK)
}


func (c *RestController) DeleteDevice(writer http.ResponseWriter, request *http.Request, dic *di.Container) {
	vars := mux.Vars(request)
	id := vars[sdkCommon.IdVar]
	correlationID := request.Header.Get(sdkCommon.CorrelationHeaderKey)

	device, ok := cache.Devices().ForId(id)
	if ok {
		c.LoggingClient.Debug(fmt.Sprintf("Handler - stopping AutoEvents for updated device %s", device.Name))
		autoevent.GetManager().StopForDevice(device.Name)
	}

	err := cache.Devices().Remove(id)
	if err == nil {
		c.LoggingClient.Info(fmt.Sprintf("Removed device: %s", device.Name))
	} else {
		c.LoggingClient.Error(fmt.Sprintf("Couldn't remove device %s: %v", device.Name, err.Error()))
		// TODO 500
	}

	driver := container.ProtocolDriverFrom(dic.Get)
	err = driver.RemoveDevice(device.Name, device.Protocols)
	if err == nil {
		c.LoggingClient.Debug(fmt.Sprintf("Invoked driver.RemoveDevice callback for %s", device.Name))
	} else {
		c.LoggingClient.Error(fmt.Sprintf("Invoked driver.RemoveDevice callback failed for %s: %v", device.Name, err.Error()))
		// TODO 500
	}

	// TODO 200
	writer.WriteHeader(http.StatusOK)
	_, err := writer.Write([]byte(correlationID)) {

	}

}