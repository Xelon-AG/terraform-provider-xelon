package device

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStructure_expandNetwork(t *testing.T) {
	type testCase struct {
		input    []interface{}
		expected *Network
	}
	tests := map[string]testCase{
		"nil": {
			input:    nil,
			expected: nil,
		},
		"empty": {
			input:    []interface{}{},
			expected: nil,
		},
		"valid": {
			input: []interface{}{0: map[string]interface{}{
				"ipv4_address_id":    1,
				"id":                 2,
				"nic_controller_key": 3,
				"nic_key":            4,
				"nic_number":         5,
				"nic_unit":           6,
			}},
			expected: &Network{
				IPAddressID:      1,
				NetworkID:        2,
				NICControllerKey: 3,
				NICKey:           4,
				NICNumber:        5,
				NICUnit:          6,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			actual := ExpandNetwork(test.input)
			assert.Equal(t, test.expected, actual)
		})
	}
}
