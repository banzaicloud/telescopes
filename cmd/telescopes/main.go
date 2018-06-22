// Package main Cluster Recommender.
//
// This project can be used to recommend instance type groups on different cloud providers consisting of regular and spot/preemptible instances.
// The main goal is to provide and continuously manage a cost-effective but still stable cluster layout that's built up from a diverse set of regular and spot instances.
//
//     Schemes: http, https
//     BasePath: /api/v1
//     Version: 0.0.1
//     License: Apache 2.0 http://www.apache.org/licenses/LICENSE-2.0.html
//     Contact: Banzai Cloud<info@banzaicloud.com>
//
// swagger:meta
package main

import (
	"net/url"
	"os"
	"strings"

	"github.com/banzaicloud/telescopes/pkg/productinfo-client/client"
	"github.com/go-openapi/strfmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/banzaicloud/telescopes/internal/app/telescopes/api"
	"github.com/banzaicloud/telescopes/pkg/recommender"
	"github.com/gin-gonic/gin"
	httptransport "github.com/go-openapi/runtime/client"
	flag "github.com/spf13/pflag"
)

const (
	// the list of flags supported by the application
	// these constants can be used to retrieve the passed in values or defaults via viper
	logLevelFlag         = "log-level"
	listenAddressFlag    = "listen-address"
	productInfoFlag      = "productinfo-address"
	devModeFlag          = "dev-mode"
	tokenSigningKeyFlag  = "token-signing-key"
	tokenSigningKeyAlias = "tokensigningkey"
	vaultAddrAlias       = "vault_addr"
	vaultAddrFlag        = "vault-address"
	helpFlag             = "help"

	cfgAppRole     = "telescopes-app-role"
	defaultAppRole = "telescopes"
)

var (
	// env vars required by the application
	cfgEnvVars = []string{tokenSigningKeyFlag, vaultAddrFlag}
	//addressRegex, _ = regexp.Compile("^((http[s]?):\\/)?\\/?([^:\\/\\s]+)((\\/\\w+)*\\/)([\\w\\-\\.]+[^#?\\s]+)(.*)$")
)

// defineFlags defines supported flags and makes them available for viper
func defineFlags() {
	flag.String(logLevelFlag, "info", "log level")
	flag.String(listenAddressFlag, ":9090", "the address where the server listens to HTTP requests.")
	flag.String(productInfoFlag, "http://localhost:9090/api/v1", "the address of the Product Info service to retrieve attribute and pricing info [format=scheme://host:port/basepath]")
	flag.Bool(devModeFlag, false, "development mode, if true token based authentication is disabled, false by default")
	flag.String(tokenSigningKeyFlag, "", "The token signing key for the authentication process")
	flag.String(vaultAddrFlag, "", "The vault address for authentication token management")
	flag.Bool(helpFlag, false, "print usage")
}

// bindFlags binds parsed flags into viper
func bindFlags() {
	flag.Parse()
	viper.BindPFlags(flag.CommandLine)
}

// setLogLevel sets the log level
func setLogLevel() {
	parsedLevel, err := log.ParseLevel(viper.GetString("log-level"))
	if err != nil {
		log.WithError(err).Warnf("Couldn't parse log level, using default: %s", log.GetLevel())
	} else {
		log.SetLevel(parsedLevel)
		log.Debugf("Set log level to %s", parsedLevel)
	}
}
func init() {

	// describe the flags for the application
	defineFlags()

	// all the flegs should be referenced through viper after this call
	// flags are available through the entire application via viper
	bindFlags()

	// handle log level
	setLogLevel()

	// set configuration defaults
	viper.SetDefault(cfgAppRole, defaultAppRole)

}

// ensureCfg ensures that the application configuration is available
// currently this only refers to configuration as environment variable
// to be extended for app critical entries (flags, config files ...)
func ensureCfg() {

	for _, envVar := range cfgEnvVars {
		// bind the env var
		viper.BindEnv(envVar)

		// read the env var value
		if nil == viper.Get(envVar) {
			flag.Usage()
			log.Fatalf("application is missing configuration: %s", envVar)
		}
	}

	// translating flags to aliases for supporting legacy env vars
	switchFlagsToAliases()

}

// switchFlagsToAliases sets the environment variables required by legacy components from application flags
// todo investigate if there's a better way for this
func switchFlagsToAliases() {
	// vault signing token hack / need to support legacy components (vault, auth)
	os.Setenv(strings.ToUpper(vaultAddrAlias), viper.GetString(vaultAddrFlag))
	os.Setenv(strings.ToUpper(tokenSigningKeyAlias), viper.GetString(tokenSigningKeyFlag))
	viper.BindEnv(vaultAddrAlias)
	viper.BindEnv(tokenSigningKeyAlias)
	log.Debugf("%s : %s", vaultAddrFlag, viper.Get(vaultAddrFlag))
	log.Debugf("%s : %s", tokenSigningKeyFlag, viper.Get(tokenSigningKeyFlag))
	log.Debugf("%s : %s", vaultAddrAlias, viper.Get(vaultAddrAlias))
	log.Debugf("%s : %s", tokenSigningKeyAlias, viper.Get(tokenSigningKeyAlias))
}

func main() {

	if viper.GetBool(helpFlag) {
		flag.Usage()
		return
	}

	ensureCfg()

	url := parseProductInfoAddress()
	transport := httptransport.New(url.Host, url.Path, []string{url.Scheme})
	pc := client.New(transport, strfmt.Default)

	engine, err := recommender.NewEngine(pc)
	quitOnError("failed to start telescopes", err)

	// configure the gin validator
	err = api.ConfigureValidator(pc)
	quitOnError("failed to start telescopes", err)

	routeHandler := api.NewRouteHandler(engine)

	// new default gin engine (recovery, logger middleware)
	router := gin.Default()

	// enable authentication if not dev-mode
	if !viper.GetBool(devModeFlag) {
		log.Debug("enable authentication")
		signingKey := viper.GetString(tokenSigningKeyAlias)
		appRole := viper.GetString(cfgAppRole)

		routeHandler.EnableAuth(router, appRole, signingKey)
	}

	log.Info("Initialized gin router")
	routeHandler.ConfigureRoutes(router)
	log.Info("Configured routes")

	router.Run(viper.GetString(listenAddressFlag))
}

func parseProductInfoAddress() *url.URL {
	productInfoAddress := viper.GetString(productInfoFlag)
	u, err := url.ParseRequestURI(productInfoAddress)
	if err != nil {
		log.Fatalf("%s is not a valid URI", productInfoFlag)
	}
	return u
}

func quitOnError(msg string, err error) {
	if err != nil {
		log.Errorf("%s : %s", msg, err.Error())
		flag.Usage()
		os.Exit(-1)
	}
}
