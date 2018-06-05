package azure

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAzure_Teszting(t *testing.T) {
	tests := []struct {
		name  string
		check func(ok bool)
	}{
		{
			name: "successful",
			check: func(ok bool) {
				assert.True(t, ok)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.check(teszting())

		})
	}
}
