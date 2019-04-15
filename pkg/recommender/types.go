// Copyright © 2019 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package recommender

const (
	// vm types - regular and ondemand means the same, they are both accepted on the API
	Regular  = "regular"
	Ondemand = "ondemand"
	Spot     = "spot"
	// Memory represents the memory attribute for the recommender
	Memory = "memory"
	// Cpu represents the cpu attribute for the recommender
	Cpu = "cpu"

	// nodepool roles
	Master = "master"
	Worker = "worker"

	RecommenderErrorTag = "recommender"
)

// ClusterRecommender is the main entry point for cluster recommendation
type ClusterRecommender interface {

	// RecommendCluster performs recommendation based on the provided arguments
	RecommendCluster(provider string, service string, region string, req ClusterRecommendationReq, layoutDesc []NodePoolDesc) (*ClusterRecommendationResp, error)

	// RecommendClusterScaleOut performs recommendation for an existing layout's scale out
	RecommendClusterScaleOut(provider string, service string, region string, req ClusterScaleoutRecommendationReq) (*ClusterRecommendationResp, error)

	// RecommendClusters performs recommendations
	RecommendClusters(req Request) (map[string][]*ClusterRecommendationResp, error)
}

type VmRecommender interface {
	RecommendVms(provider string, vms []VirtualMachine, attr string, req ClusterRecommendationReq, layout []NodePool) ([]VirtualMachine, []VirtualMachine, error)

	FindVmsWithAttrValues(attr string, req ClusterRecommendationReq, layoutDesc []NodePoolDesc, allProducts []VirtualMachine) ([]VirtualMachine, error)
}

type NodePoolRecommender interface {
	RecommendNodePools(attr string, req ClusterRecommendationReq, layout []NodePool, odVms []VirtualMachine, spotVms []VirtualMachine) []NodePool
}

// ClusterRecommendationReq encapsulates the recommendation input data
// swagger:parameters recommendClusterSetup
type ClusterRecommendationReq struct {
	// Total number of CPUs requested for the cluster
	SumCpu float64 `json:"sumCpu" binding:"min=1"`
	// Total memory requested for the cluster (GB)
	SumMem float64 `json:"sumMem" binding:"min=1"`
	// Minimum number of nodes in the recommended cluster
	MinNodes int `json:"minNodes,omitempty" binding:"min=1,ltefield=MaxNodes"`
	// Maximum number of nodes in the recommended cluster
	MaxNodes int `json:"maxNodes,omitempty"`
	// If true, recommended instance types will have a similar size
	SameSize bool `json:"sameSize,omitempty"`
	// Percentage of regular (on-demand) nodes in the recommended cluster
	OnDemandPct int `json:"onDemandPct,omitempty" binding:"min=0,max=100"`
	// Availability zones that the cluster should expand to
	Zones []string `json:"zones,omitempty"`
	// Total number of GPUs requested for the cluster
	SumGpu int `json:"sumGpu,omitempty"`
	// Are burst instances allowed in recommendation
	AllowBurst *bool `json:"allowBurst,omitempty"`
	// NetworkPerf specifies the network performance category
	NetworkPerf *string `json:"networkPerf" binding:"omitempty,networkPerf"`
	// Excludes is a blacklist - a slice with vm types to be excluded from the recommendation
	Excludes []string `json:"excludes,omitempty"`
	// Includes is a whitelist - a slice with vm types to be contained in the recommendation
	Includes []string `json:"includes,omitempty"`
	// AllowOlderGen allow older generations of virtual machines (applies for EC2 only)
	AllowOlderGen *bool `json:"allowOlderGen,omitempty"`
	// Category specifies the virtual machine category
	Category []string `json:"category,omitempty"`
}

// ClustersRecommendationReq encapsulates the recommendation input data
// swagger:parameters recommendClustersSetup
type ClustersRecommendationReq struct {
	// in:body
	Request Request `json:"request"`
}
type Request struct {
	Providers  []Provider               `json:"providers" binding:"required"`
	Continents []string                 `json:"continents" binding:"required"`
	Request    ClusterRecommendationReq `json:"request" binding:"required"`
	// Maximum number of response per service
	RespPerService int `json:"respPerService" binding:"required"`
}
type Provider struct {
	Provider string   `json:"provider"`
	Services []string `json:"services"`
}

// ClusterScaleoutRecommendationReq encapsulates the recommendation input data
// swagger:parameters recommendClusterScaleOut
type ClusterScaleoutRecommendationReq struct {
	// Total desired number of CPUs in the cluster after the scale out
	DesiredCpu float64 `json:"desiredCpu" binding:"min=1"`
	// Total desired memory (GB) in the cluster after the scale out
	DesiredMem float64 `json:"desiredMem" binding:"min=1"`
	// Total desired number of GPUs in the cluster after the scale out
	DesiredGpu int `json:"desiredGpu" binding:"min=0"`
	// Percentage of regular (on-demand) nodes among the scale out nodes
	OnDemandPct int `json:"onDemandPct,omitempty" binding:"min=0,max=100"`
	// Availability zones to be included in the recommendation
	Zones []string `json:"zones,omitempty"`
	// Excludes is a blacklist - a slice with vm types to be excluded from the recommendation
	Excludes []string `json:"excludes,omitempty"`
	// Description of the current cluster layout
	// in:body
	ActualLayout []NodePoolDesc `json:"actualLayout" binding:"required"`
}

type NodePoolDesc struct {
	// Instance type of VMs in the node pool
	InstanceType string `json:"instanceType" binding:"required"`
	// Signals that the node pool consists of regular or spot/preemptible instance types
	VmClass string `json:"vmClass" binding:"required"`
	// Number of VMs in the node pool
	SumNodes int `json:"sumNodes" binding:"required"`
	// TODO: AZ?
	// Zones []string `json:"zones,omitempty" binding:"dive,zone"`
}

func (n *NodePoolDesc) GetVmClass() string {
	switch n.VmClass {
	case Regular, Spot:
		return n.VmClass
	case Ondemand:
		return Regular
	default:
		return Spot
	}
}

// ClusterRecommendationResp encapsulates recommendation result data
// swagger:model RecommendationResponse
type ClusterRecommendationResp struct {
	// The cloud provider
	Provider string `json:"provider"`
	// Provider's service
	Service string `json:"service"`
	// Service's region
	Region string `json:"region"`
	// Availability zones in the recommendation - a multi-zone recommendation means that all node pools should expand to all zones
	Zones []string `json:"zones,omitempty"`
	// Recommended node pools
	NodePools []NodePool `json:"nodePools"`
	// Accuracy of the recommendation
	Accuracy ClusterRecommendationAccuracy `json:"accuracy"`
}

// NodePool represents a set of instances with a specific vm type
type NodePool struct {
	// Recommended virtual machine type
	VmType VirtualMachine `json:"vm"`
	// Recommended number of nodes in the node pool
	SumNodes int `json:"sumNodes"`
	// Specifies if the recommended node pool consists of regular or spot/preemptible instance types
	VmClass string `json:"vmClass"`
	// Role in the cluster, eg. master or worker
	Role string `json:"role"`
}

// PoolPrice calculates the price of the pool
func (n *NodePool) PoolPrice() float64 {
	var sum = float64(0)
	switch n.VmClass {
	case Regular:
		sum = float64(n.SumNodes) * n.VmType.OnDemandPrice
	case Spot:
		sum = float64(n.SumNodes) * n.VmType.AvgPrice
	}
	return sum
}

// GetSum gets the total value for the given attribute per pool
func (n NodePool) GetSum(attr string) float64 {
	return float64(n.SumNodes) * n.VmType.GetAttrValue(attr)
}

// ClusterRecommendationAccuracy encapsulates recommendation accuracy
type ClusterRecommendationAccuracy struct {
	// The summarised amount of memory in the recommended cluster
	RecMem float64 `json:"memory"`
	// Number of recommended cpus
	RecCpu float64 `json:"cpu"`
	// Number of recommended nodes
	RecNodes int `json:"nodes"`
	// Availability zones in the recommendation
	RecZone []string `json:"zone,omitempty"`
	// Amount of regular instance type prices in the recommended cluster
	RecRegularPrice float64 `json:"regularPrice"`
	// Number of regular instance type in the recommended cluster
	RecRegularNodes int `json:"regularNodes"`
	// Amount of spot instance type prices in the recommended cluster
	RecSpotPrice float64 `json:"spotPrice"`
	// Number of spot instance type in the recommended cluster
	RecSpotNodes int `json:"spotNodes"`
	// Total price in the recommended cluster
	RecTotalPrice float64 `json:"totalPrice"`
}

// VirtualMachine describes an instance type
type VirtualMachine struct {
	// Instance type category
	Category string
	// Instance type
	Type string `json:"type"`
	// Average price of the instance (differs from on demand price in case of spot or preemptible instances)
	AvgPrice float64 `json:"avgPrice"`
	// Regular price of the instance type
	OnDemandPrice float64 `json:"onDemandPrice"`
	// Number of CPUs in the instance type
	Cpus float64 `json:"cpusPerVm"`
	// Available memory in the instance type (GB)
	Mem float64 `json:"memPerVm"`
	// Number of GPUs in the instance type
	Gpus float64 `json:"gpusPerVm"`
	// Burst signals a burst type instance
	Burst bool `json:"burst"`
	// NetworkPerf holds the network performance
	NetworkPerf string `json:"networkPerf"`
	// NetworkPerfCat holds the network performance category
	NetworkPerfCat string `json:"networkPerfCategory"`
	// CurrentGen the vm is of current generation
	CurrentGen bool `json:"currentGen"`
	// Zones
	Zones []string `json:"zones"`
}

func (v *VirtualMachine) GetAttrValue(attr string) float64 {
	switch attr {
	case Cpu:
		return v.Cpus
	case Memory:
		return v.Mem
	default:
		return 0
	}
}
