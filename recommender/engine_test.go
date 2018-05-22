package recommender

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type dummyVmRegistry struct {
	// implement the interface
	VmRegistry
}

func TestNewEngine(t *testing.T) {

	tests := []struct {
		name         string
		vmRegistries map[string]VmRegistry
		checker      func(engine *Engine, err error)
	}{
		{
			name:         "engine successfully created",
			vmRegistries: map[string]VmRegistry{"ec2": dummyVmRegistry{}},
			checker: func(engine *Engine, err error) {
				assert.Nil(t, err, "should not get error ")
				assert.NotNil(t, engine, "the engine should not be nil")
			},
		},
		{
			name:         "engine creation fails when registries is nil",
			vmRegistries: nil,
			checker: func(engine *Engine, err error) {
				assert.Nil(t, engine, "the engine should be nil")
				assert.NotNil(t, err, "the error shouldn't be nil")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.checker(NewEngine(test.vmRegistries))

		})
	}
}

func TestEngine_RecommendCluster(t *testing.T) {
	//tests := []struct {
	//	name         string
	//	vmRegistries map[string]VmRegistry
	//	request      ClusterRecommendationReq
	//	provider     string
	//	region       string
	//	check        func(response *ClusterRecommendationResp, err error)
	//}{
	//	{
	//		name:         "cluster recommendation success",
	//		vmRegistries: map[string]VmRegistry{"ec2": dummyVmRegistry{}},
	//		request:      ClusterRecommendationReq{},
	//		provider:     "ec2",
	//		region:       "us",
	//
	//		check: func(response *ClusterRecommendationResp, err error) {
	//			assert.Nil(t, err, "should not get error when recommending")
	//
	//		},
	//	},
	//}
	//for _, test := range tests {
	//	t.Run(test.name, func(t *testing.T) {
	//		engine, err := NewEngine(test.vmRegistries)
	//		assert.Nil(t, err, "the engine couldn't be created")
	//
	//		test.check(engine.RecommendCluster(test.provider, test.region, test.request))
	//
	//	})
	//}
}
