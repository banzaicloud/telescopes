package productinfo

const (
	// network performance of vm-s
	NTW_LOW    = "low"
	NTW_MEDIUM = "medium"
	NTW_HIGH   = "high"
)

// NetworkMapper operations related  to mapping between virtual machines to network performance categories
type NetworkMapper interface {
	// NetworkForMachine gets the network performance category for the given
	NetworkPerf(vm VmInfo) (string, error)
}
