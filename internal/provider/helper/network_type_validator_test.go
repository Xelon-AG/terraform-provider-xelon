package helper

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

func TestNetworkTypeValidator(t *testing.T) {
	t.Parallel()

	type testCase struct {
		request             validator.StringRequest
		requiredExpressions path.Expressions
		expectedErrors      int
	}
	tests := map[string]testCase{
		"base-lan": {
			request: validator.StringRequest{
				ConfigValue:    types.StringValue("LAN"),
				Path:           path.Root("network_type"),
				PathExpression: path.MatchRoot("network_type"),
				Config: tfsdk.Config{
					Schema: schema.Schema{
						Attributes: map[string]schema.Attribute{
							"network_type": schema.StringAttribute{},
							"network":      schema.StringAttribute{},
						},
					},
					Raw: tftypes.NewValue(tftypes.Object{
						AttributeTypes: map[string]tftypes.Type{
							"network_type": tftypes.String,
							"network":      tftypes.String,
						},
					}, map[string]tftypes.Value{
						"network_type": tftypes.NewValue(tftypes.String, "LAN"),
						"network":      tftypes.NewValue(tftypes.String, "10.0.0.0"),
					}),
				},
			},
			requiredExpressions: path.Expressions{
				path.MatchRoot("network"),
			},
			expectedErrors: 0,
		},
		"base-wan": {
			request: validator.StringRequest{
				ConfigValue:    types.StringValue("WAN"),
				Path:           path.Root("network_type"),
				PathExpression: path.MatchRoot("network_type"),
				Config: tfsdk.Config{
					Schema: schema.Schema{
						Attributes: map[string]schema.Attribute{
							"network_type": schema.StringAttribute{},
							"subnet_size":  schema.Int64Attribute{},
						},
					},
					Raw: tftypes.NewValue(tftypes.Object{
						AttributeTypes: map[string]tftypes.Type{
							"network_type": tftypes.String,
							"subnet_size":  tftypes.Number,
						},
					}, map[string]tftypes.Value{
						"network_type": tftypes.NewValue(tftypes.String, "WAN"),
						"subnet_size":  tftypes.NewValue(tftypes.Number, 29),
					}),
				},
			},
			requiredExpressions: path.Expressions{
				path.MatchRoot("subnet_size"),
			},
			expectedErrors: 0,
		},
		"error-missing-one": {
			request: validator.StringRequest{
				ConfigValue:    types.StringValue("LAN"),
				Path:           path.Root("network_type"),
				PathExpression: path.MatchRoot("network_type"),
				Config: tfsdk.Config{
					Schema: schema.Schema{
						Attributes: map[string]schema.Attribute{
							"network_type": schema.StringAttribute{},
							"network":      schema.StringAttribute{},
						},
					},
					Raw: tftypes.NewValue(tftypes.Object{
						AttributeTypes: map[string]tftypes.Type{
							"network_type": tftypes.String,
							"network":      tftypes.String,
						},
					}, map[string]tftypes.Value{
						"network_type": tftypes.NewValue(tftypes.String, "LAN"),
						"network":      tftypes.NewValue(tftypes.String, nil),
					}),
				},
			},
			requiredExpressions: path.Expressions{
				path.MatchRoot("network"),
			},
			expectedErrors: 1,
		},
		"error-missing-two": {
			request: validator.StringRequest{
				ConfigValue:    types.StringValue("LAN"),
				Path:           path.Root("network_type"),
				PathExpression: path.MatchRoot("network_type"),
				Config: tfsdk.Config{
					Schema: schema.Schema{
						Attributes: map[string]schema.Attribute{
							"network_type": schema.StringAttribute{},
							"network":      schema.StringAttribute{},
							"subnet_size":  schema.Int64Attribute{},
						},
					},
					Raw: tftypes.NewValue(tftypes.Object{
						AttributeTypes: map[string]tftypes.Type{
							"network_type": tftypes.String,
							"network":      tftypes.String,
							"subnet_size":  tftypes.Number,
						},
					}, map[string]tftypes.Value{
						"network_type": tftypes.NewValue(tftypes.String, "LAN"),
						"network":      tftypes.NewValue(tftypes.String, nil),
						"subnet_size":  tftypes.NewValue(tftypes.Number, nil),
					}),
				},
			},
			requiredExpressions: path.Expressions{
				path.MatchRoot("network"),
				path.MatchRoot("subnet_size"),
			},
			expectedErrors: 2,
		},
		"invalid-network-type": {
			request: validator.StringRequest{
				ConfigValue:    types.StringValue("AAA"),
				Path:           path.Root("network_type"),
				PathExpression: path.MatchRoot("network_type"),
				Config:         tfsdk.Config{},
			},
			expectedErrors: 1,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			response := &validator.StringResponse{}
			v := NetworkTypeRequires(test.request.ConfigValue.ValueString(), test.requiredExpressions...)

			v.ValidateString(context.TODO(), test.request, response)

			assert.Equal(t, test.expectedErrors, response.Diagnostics.ErrorsCount())

			fmt.Println(response.Diagnostics)
		})
	}
}
