package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

func TestDeviceHasSSHKey(t *testing.T) {
	t.Parallel()

	sshKeys := []xelon.SSHKey{
		{ID: "ssh-key-1", Name: "first"},
		{ID: "ssh-key-2", Name: "second"},
	}

	tests := map[string]struct {
		sshKeys  []xelon.SSHKey
		sshKeyID string
		expected bool
	}{
		"listed key":         {sshKeys: sshKeys, sshKeyID: "ssh-key-2", expected: true},
		"unlisted key":       {sshKeys: sshKeys, sshKeyID: "ssh-key-3", expected: false},
		"no keys on device":  {sshKeys: nil, sshKeyID: "ssh-key-1", expected: false},
		"matched by id only": {sshKeys: sshKeys, sshKeyID: "first", expected: false},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.expected, DeviceHasSSHKey(test.sshKeys, test.sshKeyID))
		})
	}
}
