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

import (
	"strings"

	"github.com/banzaicloud/telescopes/internal/platform/log"
	"github.com/banzaicloud/telescopes/internal/platform/metrics"
	"github.com/goph/emperror"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Config holds any kind of configuration that comes from the outside world and
// is necessary for running the application.
type Config struct {
	// Meaningful values are recommended (eg. production, development, staging, release/123, etc)
	Environment string

	// Turns on some debug functionality
	Debug bool

	Metrics metrics.Config

	// Log configuration
	Log log.Config
}

// defineFlags defines supported flags and makes them available for viper
func defineFlags(pf *pflag.FlagSet) {
	pf.String(logLevelFlag, "info", "log level")
	pf.String(logFormatFlag, "", "log format")
	pf.String(listenAddressFlag, ":9090", "the address where the server listens to HTTP requests.")
	pf.String(cloudInfoFlag, "https://beta.banzaicloud.io/cloudinfo/api/v1", "the address of the Cloud Info service to retrieve attribute and pricing info [format=scheme://host:port/basepath]")
	pf.Bool(devModeFlag, false, "development mode, if true token based authentication is disabled, false by default")
	pf.String(tokenSigningKeyFlag, "", "The token signing key for the authentication process")
	pf.String(vaultAddrFlag, ":8200", "The vault address for authentication token management")
	pf.Bool(helpFlag, false, "print usage")
	pf.Bool(metricsEnabledFlag, false, "internal metrics are exposed if enabled")
	pf.String(metricsAddressFlag, ":9900", "the address where internal metrics are exposed")
}

// Configure configures some defaults in the Viper instance.
func Configure(v *viper.Viper, pf *pflag.FlagSet) {
	// configure viper
	// Viper check for an environment variable

	// Application constants
	v.Set("serviceName", serviceName)

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	v.AutomaticEnv()

	// Log configuration
	v.RegisterAlias("log.format", logFormatFlag)
	v.RegisterAlias("log.level", logLevelFlag)
	v.RegisterAlias("log.noColor", "no_color")

	// Metrics
	v.RegisterAlias("metrics.enabled", metricsEnabledFlag)
	v.RegisterAlias("metrics.address", metricsAddressFlag)

	pf.Init(friendlyServiceName, pflag.ExitOnError)

	// define flags
	defineFlags(pf)

	// bind flags to viper
	if err := viper.BindPFlags(pf); err != nil {
		emperror.Panic(emperror.Wrap(err, "could not parse flags"))
	}

}
