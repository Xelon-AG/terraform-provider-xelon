package provider

import (
	"net/netip"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

func TestApplyDeviceNetworkIPAddresses(t *testing.T) {
	tests := map[string]struct {
		networks       []deviceNetworkResourceModel
		deviceNetworks []xelon.DeviceNetwork
		want           []types.String
	}{
		"populates auto-assigned ip address by network id": {
			networks: []deviceNetworkResourceModel{
				{ID: types.StringValue("net-1"), IPAddress: types.StringUnknown()},
			},
			deviceNetworks: []xelon.DeviceNetwork{
				{ID: "net-1", IPAddresses: xelon.DeviceNetworkIPAddresses{netip.MustParseAddr("10.0.0.5")}},
			},
			want: []types.String{types.StringValue("10.0.0.5")},
		},
		"preserves statically configured ip address": {
			networks: []deviceNetworkResourceModel{
				{ID: types.StringValue("net-1"), IPAddress: types.StringValue("10.0.0.9")},
			},
			deviceNetworks: []xelon.DeviceNetwork{
				{ID: "net-1", IPAddresses: xelon.DeviceNetworkIPAddresses{netip.MustParseAddr("10.0.0.9")}},
			},
			want: []types.String{types.StringValue("10.0.0.9")},
		},
		"normalizes unknown without match to null": {
			networks: []deviceNetworkResourceModel{
				{ID: types.StringValue("net-1"), IPAddress: types.StringUnknown()},
			},
			deviceNetworks: []xelon.DeviceNetwork{},
			want:           []types.String{types.StringNull()},
		},
		"ignores device networks without ip addresses": {
			networks: []deviceNetworkResourceModel{
				{ID: types.StringValue("net-1"), IPAddress: types.StringUnknown()},
			},
			deviceNetworks: []xelon.DeviceNetwork{
				{ID: "net-1", IPAddresses: xelon.DeviceNetworkIPAddresses{}},
			},
			want: []types.String{types.StringNull()},
		},
		"matches multiple networks independently": {
			networks: []deviceNetworkResourceModel{
				{ID: types.StringValue("net-1"), IPAddress: types.StringUnknown()},
				{ID: types.StringValue("net-2"), IPAddress: types.StringUnknown()},
			},
			deviceNetworks: []xelon.DeviceNetwork{
				{ID: "net-2", IPAddresses: xelon.DeviceNetworkIPAddresses{netip.MustParseAddr("172.16.0.20")}},
				{ID: "net-1", IPAddresses: xelon.DeviceNetworkIPAddresses{netip.MustParseAddr("172.16.0.10")}},
			},
			want: []types.String{types.StringValue("172.16.0.10"), types.StringValue("172.16.0.20")},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := applyDeviceNetworkIPAddresses(test.networks, test.deviceNetworks)

			if len(got) != len(test.want) {
				t.Fatalf("got %d networks, want %d", len(got), len(test.want))
			}
			for i := range got {
				if !got[i].IPAddress.Equal(test.want[i]) {
					t.Errorf("network[%d] ipv4_address = %v, want %v", i, got[i].IPAddress, test.want[i])
				}
			}
		})
	}
}
