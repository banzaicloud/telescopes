package recommender

type Ec2VmRegistry struct {
}

func (e *Ec2VmRegistry) findVmsWithCpuLimits(minCpuPerVm int, maxCpuPerVm int) ([]VirtualMachine, error) {
	// TODO: this is dummy
	vms := []VirtualMachine{
		{
			Type:          "m5.xlarge",
			OnDemandPrice: 0.192,
			AvgPrice:      0.0859,
			Cpus:          4,
			Mem:           16,
			Gpus:          0,
		},
		{
			Type:          "m4.xlarge",
			OnDemandPrice: 0.2,
			AvgPrice:      0.0642,
			Cpus:          4,
			Mem:           16,
			Gpus:          0,
		},
		{
			Type:          "t2.xlarge",
			OnDemandPrice: 0.115,
			AvgPrice:      0.0605,
			Cpus:          4,
			Mem:           16,
			Gpus:          0,
		},
		{
			Type:          "r4.xlarge",
			OnDemandPrice: 0.266,
			AvgPrice:      0.071,
			Cpus:          4,
			Mem:           30.5,
			Gpus:          0,
		},
		{
			Type:          "d2.xlarge",
			OnDemandPrice: 0.69,
			AvgPrice:      0.2205,
			Cpus:          4,
			Mem:           30.5,
			Gpus:          0,
		},
		{
			Type:          "c4.xlarge",
			OnDemandPrice: 0.199,
			AvgPrice:      0.0612,
			Cpus:          4,
			Mem:           7.5,
			Gpus:          0,
		},
		{
			Type:          "m5.2xlarge",
			OnDemandPrice: 0.384,
			AvgPrice:      0.1412,
			Cpus:          8,
			Mem:           32,
			Gpus:          0,
		},
		{
			Type:          "m4.2xlarge",
			OnDemandPrice: 0.4,
			AvgPrice:      0.1284,
			Cpus:          8,
			Mem:           32,
			Gpus:          0,
		},
		{
			Type:          "r4.2xlarge",
			OnDemandPrice: 0.532,
			AvgPrice:      0.1345,
			Cpus:          8,
			Mem:           61,
			Gpus:          0,
		},
		{
			Type:          "d2.2xlarge",
			OnDemandPrice: 1.38,
			AvgPrice:      0.441,
			Cpus:          8,
			Mem:           61,
			Gpus:          0,
		},
		{
			Type:          "t2.2xlarge",
			OnDemandPrice: 0.3712,
			AvgPrice:      0.121,
			Cpus:          8,
			Mem:           32,
			Gpus:          0,
		},
		{
			Type:          "c4.2xlarge",
			OnDemandPrice: 0.398,
			AvgPrice:      0.1313,
			Cpus:          8,
			Mem:           15,
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
