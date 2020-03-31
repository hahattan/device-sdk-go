/*******************************************************************************
 * Copyright 2018 Dell Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License
 * is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
 * or implied. See the License for the specific language governing permissions and limitations under
 * the License.
 *******************************************************************************/

package common

import (
	"github.com/edgexfoundry/go-mod-bootstrap/bootstrap/interfaces"
	bootstrapConfig "github.com/edgexfoundry/go-mod-bootstrap/config"
)

// ConfigurationStruct contains the configuration properties for the device service.
type ConfigurationStruct struct {
	// WritableInfo contains configuration settings that can be changed in the Registry .
	Writable WritableInfo
	// Clients is a map of services used by a DS.
	Clients map[string]ClientInfo
	// Logging contains logging-specific configuration settings.
	Logging LoggingInfo
	// Registry contains registry-specific settings.
	Registry RegistryInfo
	// Service contains RegistryService-specific settings.
	Service ServiceInfo
	// Device contains device-specific configuration settings.
	Device DeviceInfo
	// DeviceList is the list of pre-define Devices
	DeviceList []DeviceConfig
	// Driver is a string map contains customized configuration for the protocol driver implemented based on Device SDK
	Driver map[string]string
}

// UpdateFromRaw converts configuration received from the registry to a service-specific configuration struct which is
// then used to overwrite the service's existing configuration struct.
func (c *ConfigurationStruct) UpdateFromRaw(rawConfig interface{}) bool {
	configuration, ok := rawConfig.(*ConfigurationStruct)
	if ok {
		// Check that information was successfully read from Registry
		if configuration.Service.Port == 0 {
			return false
		}
		*c = *configuration
	}
	return ok
}

// EmptyWritablePtr returns a pointer to a service-specific empty WritableInfo struct.  It is used by the bootstrap to
// provide the appropriate structure to registry.Client's WatchForChanges().
func (c *ConfigurationStruct) EmptyWritablePtr() interface{} {
	return &WritableInfo{}
}

// UpdateWritableFromRaw converts configuration received from the registry to a service-specific WritableInfo struct
// which is then used to overwrite the service's existing configuration's WritableInfo struct.
func (c *ConfigurationStruct) UpdateWritableFromRaw(rawWritable interface{}) bool {
	writable, ok := rawWritable.(*WritableInfo)
	if ok {
		c.Writable = *writable
	}
	return ok
}

// GetBootstrap returns the configuration elements required by the bootstrap.  Currently, a copy of the configuration
// data is returned.  This is intended to be temporary -- since ConfigurationStruct drives the configuration.toml's
// structure -- until we can make backwards-breaking configuration.toml changes (which would consolidate these fields
// into an interfaces.BootstrapConfiguration struct contained within ConfigurationStruct).
func (c *ConfigurationStruct) GetBootstrap() interfaces.BootstrapConfiguration {
	clients := make(map[string]bootstrapConfig.ClientInfo)
	for k, v := range c.Clients {
		clients[k] = v.ClientInfo
	}

	return interfaces.BootstrapConfiguration{
		Clients:  clients,
		Service:  c.Service.ServiceInfo,
		Registry: c.Registry.RegistryInfo,
		Logging:  c.Logging.LoggingInfo,
	}
}

// GetLogLevel returns the current ConfigurationStruct's log level.
func (c *ConfigurationStruct) GetLogLevel() string {
	return c.Writable.LogLevel
}

// GetRegistryInfo gets the config.RegistryInfo field from the ConfigurationStruct.
func (c *ConfigurationStruct) GetRegistryInfo() bootstrapConfig.RegistryInfo {
	return bootstrapConfig.RegistryInfo{
		Host: c.Registry.Host,
		Port: c.Registry.Port,
		Type: c.Registry.Type,
	}
}
