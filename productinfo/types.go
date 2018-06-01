package productinfo

// NetworkMapper operations related  to mapping between virtual machines to network performance categories
type NetworkMapper interface {
	// NetworkForMachine gets the network performance category for the given
	NetworkPerf(vm VmInfo) (string, error)
}
