package recommender

type Ec2VmRegistry struct {
}

func (e *Ec2VmRegistry) findVmsWithCpuLimits(minCpuPerVm int, maxCpuPerVm int) ([]VirtualMachine, error) {
	// TODO: this is dummy
	vms := []VirtualMachine{
		{
			Type:          "m5.xlarge",
			OnDemandPrice: 0.192,
			AvgPrice:      0.192,
			Cpus:          4,
			Mem:           16,
			Gpus:          0,
		},
		{
			Type:          "r4.xlarge",
			OnDemandPrice: 0.266,
			AvgPrice:      0.07,
			Cpus:          4,
			Mem:           30.5,
			Gpus:          0,
		},
	}
	return vms, nil
}

func (e *Ec2VmRegistry) findNearestCpuUnit(base int, larger bool) (int, error) {
	// TODO: this is dummy
	if larger {
		return 16, nil
	}
	return 8, nil
}
