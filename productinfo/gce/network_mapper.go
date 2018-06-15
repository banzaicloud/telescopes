package gce

import (
	"github.com/banzaicloud/telescopes/productinfo"
)

var (
	// TODO
	NtwPerfMap = map[string][]string{
		productinfo.NTW_LOW:    {"Low"},
		productinfo.NTW_MEDIUM: {"Moderate"},
		productinfo.NTW_HIGH:   {""},
	}
)

// GceNetworkMapper module object for handling Google Cloud specific VM to Networking capabilities mapping
type GceNetworkMapper struct {
}

// newGceNetworkMapper initializes the network performance mapper struct
func newGceNetworkMapper() *GceNetworkMapper {
	return &GceNetworkMapper{}
}

// MapNetworkPerf maps the network performance of the gce instance to the category supported by telescopes
func (nm *GceNetworkMapper) MapNetworkPerf(vm productinfo.VmInfo) (string, error) {
	return NtwPerfMap[productinfo.NTW_MEDIUM][0], nil
}
