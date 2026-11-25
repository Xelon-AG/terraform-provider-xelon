package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

func TestDataSourceXelonFirewall_NameLookup(t *testing.T) {
	tests := map[string]struct {
		firewalls          []xelon.Firewall
		expectedMatchCount int
		expectedID         string
	}{
		"ExactMatch": {
			firewalls: []xelon.Firewall{
				{ID: "firewall-1", Name: "frontend"},
			},
			expectedMatchCount: 1,
			expectedID:         "firewall-1",
		},
		"PartialOnlyNotFound": {
			firewalls: []xelon.Firewall{
				{ID: "firewall-1", Name: "frontend-production"},
			},
		},
		"CaseSensitive": {
			firewalls: []xelon.Firewall{
				{ID: "firewall-1", Name: "Frontend"},
			},
		},
		"ExactMatchAmongPartialNames": {
			firewalls: []xelon.Firewall{
				{ID: "firewall-1", Name: "frontend-production"},
				{ID: "firewall-2", Name: "frontend"},
				{ID: "firewall-3", Name: "frontend-old"},
			},
			expectedMatchCount: 1,
			expectedID:         "firewall-2",
		},
		"DuplicateExactNames": {
			firewalls: []xelon.Firewall{
				{ID: "firewall-1", Name: "frontend"},
				{ID: "firewall-2", Name: "frontend"},
			},
			expectedMatchCount: 2,
		},
		"CountsAllExactMatches": {
			firewalls: []xelon.Firewall{
				{ID: "firewall-1", Name: "frontend"},
				{ID: "firewall-2", Name: "frontend-production"},
				{ID: "firewall-3", Name: "frontend"},
				{ID: "firewall-4", Name: "frontend"},
			},
			expectedMatchCount: 3,
		},
		"EmptyList": {},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			firewalls := func(yield func(xelon.Firewall, *xelon.Response) bool) {
				for _, firewall := range test.firewalls {
					if !yield(firewall, nil) {
						return
					}
				}
			}

			firewall, matchCount := findFirewallByName(firewalls, "frontend")

			require.Equal(t, test.expectedMatchCount, matchCount)
			if matchCount == 0 {
				assert.Nil(t, firewall)
				return
			}

			require.NotNil(t, firewall)
			if matchCount == 1 {
				assert.Equal(t, test.expectedID, firewall.ID)
			}
		})
	}
}
