package productinfo

var (
	// network performance of vm-s
	NTW_LOW    = "low"
	NTW_MEDIUM = "medium"
	NTW_HIGH   = "high"
)

// NetworkPerfMapper operations related  to mapping between virtual machines to network performance categories
type NetworkPerfMapper interface {
	// MapNetworkPerf gets the network performance category for the given
	MapNetworkPerf(vm VmInfo) (string, error)
}
