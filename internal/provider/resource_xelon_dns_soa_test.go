package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

func TestResourceXelonDNSSOA_Model_FromAPI(t *testing.T) {
	soa := &xelon.DNSSOA{
		AdminEmail:   "support@cloudns.net",
		Expire:       1209600,
		PrimaryNS:    "ns1.xdns.cloud",
		Refresh:      7200,
		Retry:        1800,
		SerialNumber: 2026070301,
		TTL:          3600,
	}
	expectedModel := dnsSOAResourceModel{
		AdminEmail:        types.StringValue("support@cloudns.net"),
		Expire:            types.Int64Value(1209600),
		ID:                types.StringValue("dns-zone-1"),
		PrimaryNameserver: types.StringValue("ns1.xdns.cloud"),
		Refresh:           types.Int64Value(7200),
		Retry:             types.Int64Value(1800),
		TTL:               types.Int64Value(3600),
		ZoneID:            types.StringValue("dns-zone-1"),
	}

	var actualModel dnsSOAResourceModel
	actualModel.fromAPI(soa, "dns-zone-1")

	assert.Equal(t, expectedModel, actualModel)
}
