// Copyright © 2019 Banzai Cloud
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

const (
	serviceName = "telescopes"

	// friendlyServiceName is the visible name of the service.
	friendlyServiceName = "Banzai Cloud Telescopes Service"

	// the list of flags supported by the application
	// these constants can be used to retrieve the passed in values or defaults via viper
	logLevelFlag        = "log-level"
	logFormatFlag       = "log-format"
	listenAddressFlag   = "listen-address"
	cloudInfoFlag       = "cloudinfo-address"
	devModeFlag         = "dev-mode"
	tokenSigningKeyFlag = "tokensigningkey"
	vaultAddrFlag       = "vault-address"
	helpFlag            = "help"
	metricsEnabledFlag  = "metrics-enabled"
	metricsAddressFlag  = "metrics-address"

	cfgAppRole = "telescopes-app-role"
)
