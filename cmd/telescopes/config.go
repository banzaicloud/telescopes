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
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/banzaicloud/telescopes/internal/platform/log"
	"github.com/banzaicloud/telescopes/internal/platform/metrics"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// configuration holds any kind of configuration that comes from the outside world and
// is necessary for running the application.
type configuration struct {
	// Meaningful values are recommended (eg. production, development, staging, release/123, etc)
	Environment string

	// Turns on some debug functionality
	Debug bool

	Metrics metrics.Config

	// Log configuration
	Log log.Config

	// App configuration
	App struct {
		// HTTP server address
		Address string

		DevMode bool

		Vault struct {
			TokenSigningKey string
		}
	}

	Cloudinfo struct {
		Address string
	}
}

// Configure configures some defaults in the Viper instance.
func Configure(v *viper.Viper, p *pflag.FlagSet) {

	// Viper settings
	v.AddConfigPath(".")
	v.AddConfigPath(fmt.Sprintf("$%s_CONFIG_DIR/", strings.ToUpper(envPrefix)))

	// Environment variable settings
	// TODO: enable env prefix
	// v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AllowEmptyEnv(true)
	v.AutomaticEnv()

	// Application constants
	v.Set("appName", appName)

	// Global configuration
	v.SetDefault("environment", "production")
	v.SetDefault("debug", false)
	v.SetDefault("shutdownTimeout", 5*time.Second)
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		v.SetDefault("no_color", true)
	}

	// Log configuration
	p.String("log-level", "info", "log level")
	_ = v.BindPFlag("log.level", p.Lookup("log-level"))

	p.String("log-format", "json", "log format")
	_ = v.BindPFlag("log.format", p.Lookup("log-format"))

	v.RegisterAlias("log.noColor", "no_color")

	// Telescopes app address
	p.String("listen-address", ":9090", "the address where the server listens to HTTP requests.")
	_ = v.BindPFlag("app.address", p.Lookup("listen-address"))
	_ = v.BindEnv("app.address", "LISTEN_ADDRESS")

	// Cloudinfo
	p.String("cloudinfo-address", "http://localhost:9090/api/v1", "the address of the Cloud Info "+
		"service to retrieve attribute and pricing info [format=scheme://host:port/basepath]")
	_ = v.BindPFlag("cloudinfo.address", p.Lookup("cloudinfo-address"))
	_ = v.BindEnv("cloudinfo.address", "CLOUDINFO_ADDRESS")

	//operating mode
	p.Bool("dev-mode", false, "development mode, if true token based authentication is disabled, false by default")
	_ = v.BindPFlag("app.devmode", p.Lookup("dev-mode"))
	_ = v.BindEnv("app.devmode", "DEV_MODE")

	p.String("tokensigningkey", "", "The token signing key for the authentication process")
	_ = v.BindPFlag("app.vault.tokensigningkey", p.Lookup("tokensigningkey"))
	_ = v.BindEnv("app.vault.tokensigningkey", "TOKENSIGNINIGKEY")

	p.String("vault-address", ":8200", "The vault address for authentication token management")
	_ = v.BindPFlag("app.vault.address", p.Lookup("vault-address"))
	_ = v.BindEnv("app.vault.address", "VAULT_ADDRESS")

	p.Bool("metrics-enabled", false, "internal metrics are exposed if enabled")
	_ = v.BindPFlag("metrics.enabled", p.Lookup("metrics-enabled"))
	_ = v.BindEnv("metrics.enabled", "METRICS_ENABLED")

	p.String("metrics-address", ":9900", "the address where internal metrics are exposed")
	_ = v.BindPFlag("metrics.address", p.Lookup("metrics-address"))
	_ = v.BindEnv("metrics.address", "METRICS_ADDRESS")

	p.Init(friendlyAppName, pflag.ExitOnError)

}
