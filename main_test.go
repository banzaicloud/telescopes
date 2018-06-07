package main

import (
	"github.com/banzaicloud/telescopes/recommender"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

var stdin *os.File

func Test_processFlags(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		check func(val interface{})
	}{
		{
			name: "flag made available through viper",
			args: []string{
				"--log-level", "debug",
			},
			check: func(val interface{}) {
				assert.Equal(t, "debug", val)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
			// define flags
			defineFlags()
			// mock the input
			setupInputs(test.args, nil)
			test.check(viper.GetString("log-level"))

		})
	}
}
func Test_processProviderFlag(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		check func(val interface{})
	}{
		{
			name: "--provider flag properly made available through viper",
			args: []string{
				// notice the 3 ways providers may be given
				"--provider=ec2,gke", "--provider=azure", "--provider", "alibaba",
			},
			check: func(val interface{}) {
				assert.Equal(t, []string{"ec2", "gke", "azure", "alibaba"}, val)

			},
		},
		{
			name: "--provider flag default values",
			args: []string{
				// no provider flag specified
			},
			check: func(val interface{}) {
				assert.Equal(t, []string{recommender.Ec2, recommender.Gce}, val)

			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
			// define flags
			defineFlags()
			// mock the input
			setupInputs(test.args, nil)
			test.check(viper.GetStringSlice("provider"))

		})
	}
}

// setupInputs mocks out the command line argument list
func setupInputs(args []string, file *os.File) {

	// This trick allows command line flags to be be set in unit tests.
	// See https://github.com/VonC/gogitolite/commit/f656a9858cb7e948213db9564a9120097252b429
	a := os.Args[1:]
	if args != nil {
		a = args
	}
	pflag.CommandLine.Parse(a)
	viper.BindPFlags(pflag.CommandLine)

	// This enables stdin to be mocked for testing.
	if file != nil {
		stdin = file
	} else {
		stdin = os.Stdin
	}
}
