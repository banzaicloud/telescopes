package productinfo

var (
	// telescope supported network performance of vm-s

	// NTW_LOW the low network performance category
	NTW_LOW = "low"
	// NTW_MEDIUM the medium network performance category
	NTW_MEDIUM = "medium"
	// NTW_HIGH the high network performance category
	NTW_HIGH = "high"
)

// NetworkPerfMapper operations related  to mapping between virtual machines to network performance categories
type NetworkPerfMapper interface {
	// MapNetworkPerf gets the network performance category for the given
	MapNetworkPerf(vm VmInfo) (string, error)
}
