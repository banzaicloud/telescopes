package ec2

import (
	"fmt"
	"github.com/banzaicloud/telescopes/productinfo"
)

var (
	ntwPerfMap = map[string][]string{
		// available categories
		//"10 Gigabit"
		//"20 Gigabit"
		//"25 Gigabit"
		//"High"
		//"Low to Moderate"
		//"Low"
		//"Moderate"
		//"NA"
		//"Up to 10 Gigabit"
		//"Very Low"

		productinfo.NTW_LOW:    {"Very Low", "Low", "Low to Moderate"},
		productinfo.NTW_MEDIUM: {"Moderate"},
		productinfo.NTW_HIGH:   {"High", "Up to 10 Gigabit", "10 Gigabit", "20 Gigabit", "25 Gigabit"},
	}
)

// Ec2NetworkMapper module object for handling amazon sopecific VM to Networking capabilities mapping
type Ec2NetworkMapper struct {
}

func NewEc2NetworkMapper() Ec2NetworkMapper {
	return Ec2NetworkMapper{}
}

func (nm *Ec2NetworkMapper) NetworkPerf(vm productinfo.VmInfo) (string, error) {
	for perfCat, strVals := range ntwPerfMap {
		if contains(strVals, vm.NtwPerf) {
			return perfCat, nil
		}
	}
	return "", fmt.Errorf("could not determine network performance for: [%s]", vm.NtwPerf)
}

func contains(slice []string, val string) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}
