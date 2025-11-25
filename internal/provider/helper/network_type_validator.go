package helper

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = &networkTypeValidator{}

const (
	networkTypeLAN = "LAN"
	networkTypeWAN = "WAN"
)

type networkTypeValidator struct {
	networkType         string
	requiredExpressions path.Expressions
}

func (v networkTypeValidator) Description(ctx context.Context) string {
	return v.MarkdownDescription(ctx)
}

func (v networkTypeValidator) MarkdownDescription(_ context.Context) string {
	return fmt.Sprintf("value must be one of: %q", []string{networkTypeLAN, networkTypeWAN})
}

func (v networkTypeValidator) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	value := request.ConfigValue.ValueString()
	if value != networkTypeLAN && value != networkTypeWAN {
		response.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
			request.Path,
			v.Description(ctx),
			value,
		))
		return
	}

	expressions := request.PathExpression.MergeExpressions(v.requiredExpressions...)

	for _, expression := range expressions {
		matchedPaths, diags := request.Config.PathMatches(ctx, expression)
		response.Diagnostics.Append(diags...)
		// collect all errors
		if diags.HasError() {
			continue
		}

		for _, matchedPath := range matchedPaths {
			//
			var matchedPathValue attr.Value
			diags := request.Config.GetAttribute(ctx, matchedPath, &matchedPathValue)
			response.Diagnostics.Append(diags...)
			// collect all errors
			if diags.HasError() {
				continue
			}
			// delay validation until all involved attribute have a known value
			if matchedPathValue.IsUnknown() {
				return
			}
			if matchedPathValue.IsNull() && v.networkType == value {
				response.Diagnostics.Append(validatordiag.InvalidAttributeCombinationDiagnostic(
					request.Path,
					fmt.Sprintf("Attribute %q must be specified when %q is %s.", matchedPath, request.Path, v.networkType),
				))
			}
		}
	}
}

func NetworkTypeRequires(networkType string, expressions ...path.Expression) validator.String {
	return &networkTypeValidator{
		networkType:         networkType,
		requiredExpressions: expressions,
	}
}
