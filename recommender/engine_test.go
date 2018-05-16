package recommender

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewEngine(t *testing.T) {

	type dummyVmRegistry struct {
		// implement the interface
		VmRegistry
	}
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
				assert.Nil(t, engine, "the e ngine should be nil")
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
