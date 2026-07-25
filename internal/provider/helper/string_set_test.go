package helper

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSortedStringSetElements_Sorted(t *testing.T) {
	value, diags := types.SetValueFrom(context.Background(), types.StringType, []string{"role-b", "role-a"})
	require.False(t, diags.HasError())

	actual, diags := SortedStringSetElements(context.Background(), value)

	require.False(t, diags.HasError())
	assert.Equal(t, []string{"role-a", "role-b"}, actual)
}

func TestSortedStringSetElements_NullOrUnknown(t *testing.T) {
	testCases := map[string]types.Set{
		"null":    types.SetNull(types.StringType),
		"unknown": types.SetUnknown(types.StringType),
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			actual, diags := SortedStringSetElements(context.Background(), testCase)

			require.False(t, diags.HasError())
			assert.Nil(t, actual)
		})
	}
}
