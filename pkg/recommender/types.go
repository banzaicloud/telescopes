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

import (
	"math"
)

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
	RecommendCluster(provider string, service string, region string, req SingleClusterRecommendationReq, layoutDesc []NodePoolDesc) (*ClusterRecommendationResp, error)

	// RecommendClusterScaleOut performs recommendation for an existing layout's scale out
	RecommendClusterScaleOut(provider string, service string, region string, req ClusterScaleoutRecommendationReq) (*ClusterRecommendationResp, error)

	// RecommendMultiCluster performs recommendations
	RecommendMultiCluster(req MultiClusterRecommendationReq) (map[string][]*ClusterRecommendationResp, error)
}

type VmRecommender interface {
	RecommendVms(provider string, vms []VirtualMachine, attr string, req SingleClusterRecommendationReq, layout []NodePool) ([]VirtualMachine, []VirtualMachine, error)

	FindVmsWithAttrValues(attr string, req SingleClusterRecommendationReq, layoutDesc []NodePoolDesc, allProducts []VirtualMachine) ([]VirtualMachine, error)
}

type NodePoolRecommender interface {
	RecommendNodePools(attr string, req SingleClusterRecommendationReq, layout []NodePool, odVms []VirtualMachine, spotVms []VirtualMachine) []NodePool
}

// SingleClusterRecommendationReq encapsulates the recommendation input data, the filters applied is in the order IncludeSeries -> ExcludeSeries ->
// swagger:model recommendClusterRequest
type SingleClusterRecommendationReq struct {
	// Embedded struct
	ClusterRecommendationReq
	// IncludeSeries is a whitelist - a group of vm of particular series/families to be contained in the recommendation
	IncludeSeries []string `json:"includeSeries,omitempty"`
	// ExcludeSeries is a blacklist - a group of vm of particular series/families to be excluded from the recommendation
	ExcludeSeries []string `json:"excludeSeries,omitempty"`
	// IncludeTypes is a whitelist - a slice with vm types to be contained in the recommendation
	IncludeTypes []string `json:"includeTypes,omitempty"`
	// ExcludeTypes is a blacklist - a slice with vm types to be excluded from the recommendation
	ExcludeTypes []string `json:"excludeTypes,omitempty"`
	// Availability zone that the cluster should expand to
	Zone string `json:"zone,omitempty"`
}

// ClusterRecommendationReq encapsulates the recommendation input data
type ClusterRecommendationReq struct {
	// Total number of CPUs requested for the cluster
	SumCpu float64 `json:"sumCpu" binding:"min=0"`
	// Total memory requested for the cluster (GB)
	SumMem float64 `json:"sumMem" binding:"min=0"`
	// Max of CPUs requested by any Pod in the the cluster
	MinCpu *float64 `json:"minCpu,omitempty"`
	// Max of Memory requested by any Pod in the the cluster (GB)
	MinMem *float64 `json:"minMem,omitempty"`
	// Minimum number of nodes in the recommended cluster
	MinNodes int `json:"minNodes,omitempty" binding:"min=1,ltefield=MaxNodes"`
	// Maximum number of nodes in the recommended cluster
	MaxNodes int `json:"maxNodes,omitempty"`
	// If true, recommended instance types will have a similar size
	SameSize bool `json:"sameSize,omitempty"`
	// Percentage of regular (on-demand) nodes in the recommended cluster
	OnDemandPct int `json:"onDemandPct,omitempty" binding:"min=0,max=100"`
	// Total number of GPUs requested for the cluster
	SumGpu int `json:"sumGpu,omitempty"`
	// Are burst instances allowed in recommendation
	AllowBurst *bool `json:"allowBurst,omitempty"`
	// NetworkPerf specifies the network performance category
	NetworkPerf []string `json:"networkPerf" binding:"omitempty,dive,networkPerf"`
	// AllowOlderGen allow older generations of virtual machines (applies for EC2 only)
	AllowOlderGen *bool `json:"allowOlderGen,omitempty"`
	// Category specifies the virtual machine category
	Category []string `json:"category" binding:"omitempty,dive,category"`
}

// MultiClusterRecommendationReq encapsulates the recommendation input data
// swagger:model recommendMultiClusterRequest
type MultiClusterRecommendationReq struct {
	Providers  []Provider `json:"providers" binding:"required"`
	Continents []string   `json:"continents"`
	// Embedded struct
	ClusterRecommendationReq
	// Excludes is a blacklist - a slice with vm types to be excluded from the recommendation
	Excludes map[string]map[string][]string `json:"excludes,omitempty"`
	// Includes is a whitelist - a slice with vm types to be contained in the recommendation
	Includes map[string]map[string][]string `json:"includes,omitempty"`
	// Maximum number of response per service
	RespPerService int `json:"respPerService" binding:"required"`
}

type Provider struct {
	Provider string   `json:"provider"`
	Services []string `json:"services"`
}

// ClusterScaleoutRecommendationReq encapsulates the recommendation input data
// swagger:model recommendClusterScaleOutRequest
type ClusterScaleoutRecommendationReq struct {
	// Total desired number of CPUs in the cluster after the scale out
	DesiredCpu float64 `json:"desiredCpu" binding:"min=1"`
	// Total desired memory (GB) in the cluster after the scale out
	DesiredMem float64 `json:"desiredMem" binding:"min=1"`
	// Total desired number of GPUs in the cluster after the scale out
	DesiredGpu int `json:"desiredGpu" binding:"min=0"`
	// Percentage of regular (on-demand) nodes among the scale out nodes
	OnDemandPct int `json:"onDemandPct,omitempty" binding:"min=0,max=100"`
	// Availability zone to be included in the recommendation
	Zone string `json:"zone,omitempty"`
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
type ClusterRecommendationResp struct {
	// The cloud provider
	Provider string `json:"provider"`
	// Provider's service
	Service string `json:"service"`
	// Service's region
	Region string `json:"region"`
	// Availability zone in the recommendation - a multi-zone recommendation means that all node pools should expand to all zones
	Zone string `json:"zone,omitempty"`
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

//PoolPriceByClass calculates the price of the pool by class
func (n *NodePool) PoolPriceByClass(vmClass string) float64 {
	var sum = float64(0)
	switch vmClass {
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

// GetAllocSum gets the total value for the given attribute per pool
func (n NodePool) GetAllocSum(attr string) float64 {
	return float64(n.SumNodes) * n.VmType.GetAllocatableAttrValue(attr)
}

// ClusterRecommendationAccuracy encapsulates recommendation accuracy
type ClusterRecommendationAccuracy struct {
	// The summarised amount of memory in the recommended cluster
	RecMem float64 `json:"memory"`
	// Number of recommended cpus
	RecCpu float64 `json:"cpu"`
	// Number of recommended allocatable memory
	RecAllocatableMem float64 `json:"allocatableMemory"`
	// Number of recommended allocatable cpus
	RecAllocatableCpu float64 `json:"allocatableCpu"`
	// Number of recommended nodes
	RecNodes int `json:"nodes"`
	// Availability zone in the recommendation
	RecZone string `json:"zone,omitempty"`
	// Amount of regular instance type prices in the recommended cluster
	RecRegularPrice float64 `json:"regularPrice"`
	// Number of regular instance type in the recommended cluster
	RecRegularNodes int `json:"regularNodes"`
	// Amount of spot instance type prices in the recommended cluster
	RecSpotPrice float64 `json:"spotPrice"`
	// Number of spot instance type in the recommended cluster
	RecSpotNodes int `json:"spotNodes"`
	// Amount of worker instance type prices in the recommended cluster
	RecWorkerPrice float64 `json:"workerPrice"`
	// Amount of master instance type prices in the recommended cluster
	RecMasterPrice float64 `json:"masterPrice"`
	// Total price in the recommended cluster
	RecTotalPrice float64 `json:"totalPrice"`
}

// VirtualMachine describes an instance type
type VirtualMachine struct {
	// Average price of the instance (differs from on demand price in case of spot or preemptible instances)
	AvgPrice float64 `json:"avgPrice"`
	// Regular price of the instance type
	OnDemandPrice float64 `json:"onDemandPrice"`
	// Number of CPUs in the instance type
	Cpus float64 `json:"cpusPerVm"`
	// Number of allocatable CPUs in the instance type
	AllocatableCpus float64 `json:"allocatableCpusPerVm"`
	// Memory capacity in the instance type (GB)
	Mem float64 `json:"memPerVm"`
	// Allocatable memory in the instance type (GB)
	AllocatableMem float64 `json:"allocatableMemPerVm"`
	// Number of GPUs in the instance type
	Gpus float64 `json:"gpusPerVm"`
	// Burst signals a burst type instance
	Burst bool `json:"burst"`
	// CurrentGen the vm is of current generation
	CurrentGen bool `json:"currentGen"`
	// Zones
	Zones []string `json:"zones"`
	// Instance type category
	Category string `json:"category"`
	// Instance type series/family
	Series string `json:"series"`
	// Instance type
	Type string `json:"type"`
	// NetworkPerf holds the network performance
	NetworkPerf string `json:"networkPerf"`
	// NetworkPerfCat holds the network performance category
	NetworkPerfCat string `json:"networkPerfCategory"`
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

func (v *VirtualMachine) GetAllocatableAttrValue(attr string) float64 {
	switch attr {
	case Cpu:
		return v.AllocatableCpus
	case Memory:
		return v.AllocatableMem
	default:
		return 0
	}
}

func (v *VirtualMachine) ComputeAllocatableAttrValueForGKE(attr string) float64 {
	//the resources used by kube-system pods, the amount of which varies with each Kubernetes release. These system
	//pods generally occupy an additional 0.4 vCPU and 400 MiB memory on each node (values are approximate).
	//ref https://cloud.google.com/kubernetes-engine/docs/concepts/cluster-architecture

	//GKE reserves approximately 1.5 times more resources on Windows Server nodes, so the allocatable values are lower
	//than the Linux values.
	//below calculations are based on linux instances
	switch attr {
	case Cpu:
		var reservedCpus float64
		kubeSystemOverhead := 0.4

		// 6% of the first core
		if v.Cpus > 0.0 {
			reservedCpus += 0.06 * math.Min(v.Cpus, 1.0)
		}

		// 1% of the next core (up to 2 cores)
		if v.Cpus > 1.0 {
			reservedCpus += 0.01 * math.Min(v.Cpus-1.0, 1.0)
		}

		// 0.5% of the next 2 cores (up to 4 cores)
		if v.Cpus > 2.0 {
			reservedCpus += 0.005 * math.Min(v.Cpus-2.0, 2.0)
		}

		// 0.25% of any cores above 4 cores
		if v.Cpus > 4.0 {
			reservedCpus += 0.0025 * math.Min(v.Cpus-4.0, math.MaxFloat64)
		}

		return v.Cpus - reservedCpus - kubeSystemOverhead
	case Memory:
		var reservedMemory float64
		kubeSystemOverhead := 400.0 / 1024.0
		evictionThreshold := 100.0 / 1024.0

		//255 MiB of memory for machines with less than 1 GB of memory
		if v.Mem < 1.0 {
			reservedMemory += 255.0 / 1024.0
		}

		//25% of the first 4GB of memory
		if v.Mem >= 1.0 {
			reservedMemory += 0.25 * math.Min(v.Mem, 4.0)
		}

		//20% of the next 4GB of memory (up to 8GB)
		if v.Mem > 4.0 {
			reservedMemory += 0.20 * math.Min(v.Mem-4.0, 4.0)
		}

		//10% of the next 8GB of memory (up to 16GB)
		if v.Mem > 8.0 {
			reservedMemory += 0.10 * math.Min(v.Mem-8.0, 8.0)
		}

		//6% of the next 112GB of memory (up to 128GB)
		if v.Mem > 16.0 {
			reservedMemory += 0.06 * math.Min(v.Mem-16.0, 112.0)
		}

		//2% of any memory above 128GB
		if v.Mem > 128.0 {
			reservedMemory += 0.02 * math.Min(v.Mem-128.0, math.MaxFloat64)
		}

		return v.Mem - reservedMemory - evictionThreshold - kubeSystemOverhead
	default:
		return 0
	}
}
