// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2017-2018 Canonical Ltd
// Copyright (C) 2018-2020 IOTech Ltd
// Copyright (c) 2019 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/edgexfoundry/device-sdk-go/internal/common"
	"github.com/edgexfoundry/device-sdk-go/internal/controller/correlation"
	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/di"
	"github.com/edgexfoundry/go-mod-core-contracts/clients"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	contractsV2 "github.com/edgexfoundry/go-mod-core-contracts/v2"
	"github.com/gorilla/mux"
)

type RestController struct {
	LoggingClient  logger.LoggingClient
	router         *mux.Router
	reservedRoutes map[string]bool
}

func NewRestController(r *mux.Router, lc logger.LoggingClient) *RestController {
	return &RestController{
		LoggingClient:  lc,
		router:         r,
		reservedRoutes: make(map[string]bool),
	}
}

func (c *RestController) InitRestRoutes(dic *di.Container) {
	// Status
	c.addReservedRoute(common.APIPingRoute, c.statusFunc, dic).Methods(http.MethodGet)
	// Version
	c.addReservedRoute(common.APIVersionRoute, c.versionFunc, dic).Methods(http.MethodGet)
	// Command
	c.addReservedRoute(common.APIAllCommandRoute, c.commandAllFunc, dic).Methods(http.MethodGet, http.MethodPut)
	c.addReservedRoute(common.APIIdCommandRoute, c.commandFunc, dic).Methods(http.MethodGet, http.MethodPut)
	c.addReservedRoute(common.APINameCommandRoute, c.commandFunc, dic).Methods(http.MethodGet, http.MethodPut)
	// Callback
	c.addReservedRoute(common.APICallbackRoute, c.callbackFunc, dic)
	// Discovery and Transform
	c.addReservedRoute(common.APIDiscoveryRoute, c.discoveryFunc, dic).Methods(http.MethodPost)
	c.addReservedRoute(common.APITransformRoute, c.transformFunc, dic).Methods(http.MethodGet)
	// Metric and Config
	c.addReservedRoute(common.APIMetricsRoute, c.metricsFunc, dic).Methods(http.MethodGet)
	c.addReservedRoute(common.APIConfigRoute, c.configFunc, dic).Methods(http.MethodGet)

	c.InitV2RestRoutes(dic)

	c.router.Use(correlation.ManageHeader)
	c.router.Use(correlation.OnResponseComplete)
	c.router.Use(correlation.OnRequestBegin)
}

func (c *RestController) InitV2RestRoutes(dic *di.Container) {
	c.LoggingClient.Info("Registering standard v2 routes...")

	c.addReservedRoute(contractsV2.ApiPingRoute, c.Ping, dic).Methods(http.MethodGet)
	c.addReservedRoute(contractsV2.ApiVersionRoute, c.Version, dic).Methods(http.MethodGet)
	c.addReservedRoute(contractsV2.ApiConfigRoute, c.Config, dic).Methods(http.MethodGet)
	c.addReservedRoute(contractsV2.ApiMetricsRoute, c.Metrics, dic).Methods(http.MethodGet)

	c.addReservedRoute(contractsV2.ApiBase + "/callback/device", nil, dic).Methods(http.MethodPut, http.MethodPost)
	c.addReservedRoute(contractsV2.ApiBase + "/callback/device/id/{id}", c.DeleteDevice, dic).Methods(http.MethodDelete)
	c.addReservedRoute(contractsV2.ApiBase + "/callback/profile", nil, dic).Methods(http.MethodPut, http.MethodPost)
	c.addReservedRoute(contractsV2.ApiBase + "/callback/profile/id/{id}", nil, dic).Methods(http.MethodDelete)
	c.addReservedRoute(contractsV2.ApiBase + "/callback/watcher", nil, dic).Methods(http.MethodPut, http.MethodPost)
	c.addReservedRoute(contractsV2.ApiBase + "/callback/watcher/id/{id}", nil, dic).Methods(http.MethodDelete)

}

// sendResponse puts together the response packet for the V2 API
func (c *RestController) sendResponse(
	writer http.ResponseWriter,
	request *http.Request,
	api string,
	response interface{},
	statusCode int) {

	correlationID := request.Header.Get(common.CorrelationHeaderKey)

	writer.WriteHeader(statusCode)
	writer.Header().Set(common.CorrelationHeaderKey, correlationID)
	writer.Header().Set(clients.ContentType, clients.ContentTypeJSON)

	data, err := json.Marshal(response)
	if err != nil {
		c.LoggingClient.Error(fmt.Sprintf("Unable to marshal %s response", api), "error", err.Error(), clients.CorrelationHeader, correlationID)
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = writer.Write(data)
	if err != nil {
		c.LoggingClient.Error(fmt.Sprintf("Unable to write %s response", api), "error", err.Error(), clients.CorrelationHeader, correlationID)
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (c *RestController) addReservedRoute(route string, handler func(http.ResponseWriter, *http.Request, *di.Container), dic *di.Container) *mux.Route {
	c.reservedRoutes[route] = true
	return c.router.HandleFunc(
		route,
		func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), bootstrapContainer.LoggingClientInterfaceName, c.LoggingClient)
			handler(
				w,
				r.WithContext(ctx),
				dic)

		})
}

func (c *RestController) AddRoute(route string, handler func(http.ResponseWriter, *http.Request), methods ...string) error {
	if c.reservedRoutes[route] {
		return errors.New("route is reserved")
	}

	c.router.HandleFunc(
		route,
		func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), bootstrapContainer.LoggingClientInterfaceName, c.LoggingClient)
			handler(
				w,
				r.WithContext(ctx))
		}).Methods(methods...)
	c.LoggingClient.Debug("Route added", "route", route, "methods", fmt.Sprintf("%v", methods))

	return nil
}

func (c *RestController) Router() *mux.Router {
	return c.router
}
