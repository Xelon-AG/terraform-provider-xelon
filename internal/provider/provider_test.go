package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProvider_userAgent(t *testing.T) {
	t.Parallel()

	type testCase struct {
		version           string
		expectedUserAgent string
	}
	tests := map[string]testCase{
		"empty_version": {
			version:           "",
			expectedUserAgent: "terraform-provider-xelon/ (+https://registry.terraform.io/providers/Xelon-AG/xelon)",
		},
		"dev_version": {
			version:           "dev",
			expectedUserAgent: "terraform-provider-xelon/dev (+https://registry.terraform.io/providers/Xelon-AG/xelon)",
		},
		"release_version": {
			version:           "1.1.1",
			expectedUserAgent: "terraform-provider-xelon/1.1.1 (+https://registry.terraform.io/providers/Xelon-AG/xelon)",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p := &xelonProvider{version: test.version}
			actualUserAgent := p.userAgent()

			assert.Equal(t, test.expectedUserAgent, actualUserAgent)
		})
	}
}
