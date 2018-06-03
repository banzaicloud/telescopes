package ec2

import "github.com/banzaicloud/telescopes/productinfo"

var (
	ntwPerfMap = map[string][]string{
		productinfo.NTW_HIGH:   {},
		productinfo.NTW_MEDIUM: {},
		productinfo.NTW_LOW:    {},
	}
)

// Ec2NetworkMapper module object for handling amazon sopecific VM to Networking capabilities mapping
type Ec2NetworkMapper struct {
}

func NewEc2NetworkMapper() Ec2NetworkMapper {
	return Ec2NetworkMapper{}
}

func (nm *Ec2NetworkMapper) NetworkPerf(vm productinfo.VmInfo) (string, error) {
	// todo
	return productinfo.NTW_HIGH, nil
}
