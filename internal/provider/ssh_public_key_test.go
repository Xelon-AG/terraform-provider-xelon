package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSHPublicKeyType_ValueFromTerraform(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		in            tftypes.Value
		expectation   attr.Value
		expectedError string
	}{
		"true": {
			in:          tftypes.NewValue(tftypes.String, "ssh-ed25519 AAAA user@example.com"),
			expectation: newSSHPublicKeyValue("ssh-ed25519 AAAA user@example.com"),
		},
		"wrong type": {
			in:            tftypes.NewValue(tftypes.Number, 123),
			expectedError: "can't unmarshal tftypes.Number into *string, expected string",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			actual, err := sshPublicKeyType{}.ValueFromTerraform(context.Background(), test.in)
			if err != nil {
				if test.expectedError == "" {
					require.Error(t, err)
				}
				assert.Equal(t, test.expectedError, err.Error())
			}

			assert.Equal(t, test.expectation, actual)
		})
	}
}
