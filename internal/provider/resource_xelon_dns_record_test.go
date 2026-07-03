package provider

import (
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

func TestResourceXelonDNSRecord_CompositeID_Build(t *testing.T) {
	assert.Equal(t, "zone-123/456", buildDNSRecordCompositeID("zone-123", 456))
}

func TestResourceXelonDNSRecord_CompositeID_Parse(t *testing.T) {
	zoneID, recordID, err := parseDNSRecordCompositeID("zone-123/456")

	require.NoError(t, err)
	assert.Equal(t, "zone-123", zoneID)
	assert.Equal(t, int64(456), recordID)
}

func TestResourceXelonDNSRecord_CompositeID_ParseInvalid(t *testing.T) {
	testCases := []string{
		"",
		"zone-123",
		"/456",
		"zone-123/",
		"zone-123/abc",
		"zone-123/0",
		"zone-123/-1",
		"zone-123/456/extra",
	}

	for _, testCase := range testCases {
		t.Run(testCase, func(t *testing.T) {
			_, _, err := parseDNSRecordCompositeID(testCase)
			require.Error(t, err)
		})
	}
}

func TestResourceXelonDNSRecord_RecordLookup_FindByID(t *testing.T) {
	records := []xelon.DNSRecord{
		{ID: 100, Host: "www"},
		{ID: 200, Host: "api"},
	}

	record := findDNSRecordByID(records, 200)

	require.NotNil(t, record)
	assert.Equal(t, "api", record.Host)
}

func TestResourceXelonDNSRecord_RecordLookup_FindByIDMissing(t *testing.T) {
	records := []xelon.DNSRecord{
		{ID: 100, Host: "www"},
	}

	record := findDNSRecordByID(records, 200)

	assert.Nil(t, record)
}

func TestResourceXelonDNSRecord_CreatedRecordLookup_FindMatch(t *testing.T) {
	planned := &dnsRecordResourceModel{
		Content: types.StringValue("203.0.113.10"),
		Name:    types.StringValue("www"),
		TTL:     types.Int64Value(3600),
		Type:    types.StringValue("A"),
	}
	records := []xelon.DNSRecord{
		{ID: 100, Host: "api", Record: "203.0.113.20", TTL: 3600, Type: xelon.DNSRecordTypeA},
		{ID: 200, Host: "www", Record: "203.0.113.10", TTL: 3600, Type: xelon.DNSRecordTypeA},
	}

	record, matchCount := findCreatedDNSRecord(records, planned)

	require.Equal(t, 1, matchCount)
	require.NotNil(t, record)
	assert.Equal(t, 200, record.ID)
}

func TestResourceXelonDNSRecord_CreatedRecordLookup_ZeroMatches(t *testing.T) {
	planned := &dnsRecordResourceModel{
		Content: types.StringValue("203.0.113.10"),
		Name:    types.StringValue("www"),
		TTL:     types.Int64Value(3600),
		Type:    types.StringValue("A"),
	}
	records := []xelon.DNSRecord{
		{ID: 100, Host: "api", Record: "203.0.113.20", TTL: 3600, Type: xelon.DNSRecordTypeA},
	}

	record, matchCount := findCreatedDNSRecord(records, planned)

	assert.Nil(t, record)
	assert.Equal(t, 0, matchCount)
}

func TestResourceXelonDNSRecord_CreatedRecordLookup_AmbiguousMatches(t *testing.T) {
	planned := &dnsRecordResourceModel{
		Content: types.StringValue("203.0.113.10"),
		Name:    types.StringValue("www"),
		TTL:     types.Int64Value(3600),
		Type:    types.StringValue("A"),
	}
	records := []xelon.DNSRecord{
		{ID: 100, Host: "www", Record: "203.0.113.10", TTL: 3600, Type: xelon.DNSRecordTypeA},
		{ID: 200, Host: "www", Record: "203.0.113.10", TTL: 3600, Type: xelon.DNSRecordTypeA},
	}

	record, matchCount := findCreatedDNSRecord(records, planned)

	require.Equal(t, 2, matchCount)
	require.NotNil(t, record)
	assert.Equal(t, 200, record.ID)
}

func TestResourceXelonDNSRecord_SupportedV0RecordTypes(t *testing.T) {
	supportedTypes := supportedV0DNSRecordTypes()

	expectedSupportedTypes := []string{
		string(xelon.DNSRecordTypeA),
		string(xelon.DNSRecordTypeAAAA),
		string(xelon.DNSRecordTypeCNAME),
		string(xelon.DNSRecordTypeTXT),
		string(xelon.DNSRecordTypeNS),
		string(xelon.DNSRecordTypeALIAS),
		string(xelon.DNSRecordTypePTR),
	}
	for _, recordType := range expectedSupportedTypes {
		assert.Truef(t, slices.Contains(supportedTypes, recordType), "expected %s to be supported", recordType)
	}

	expectedUnsupportedTypes := []string{
		string(xelon.DNSRecordTypeMX),
		string(xelon.DNSRecordTypeSRV),
		string(xelon.DNSRecordTypeCAA),
		string(xelon.DNSRecordTypeRP),
		string(xelon.DNSRecordTypeSSHFP),
		string(xelon.DNSRecordTypeTLSA),
	}
	for _, recordType := range expectedUnsupportedTypes {
		assert.Falsef(t, slices.Contains(supportedTypes, recordType), "expected %s to be unsupported", recordType)
	}
}
