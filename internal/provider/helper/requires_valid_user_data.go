package helper

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = (*requiresValidUserData)(nil)

type requiresValidUserData struct {
	requiredExpressions path.Expressions
}

func (av requiresValidUserData) Description(ctx context.Context) string {
	return av.MarkdownDescription(ctx)
}

func (av requiresValidUserData) MarkdownDescription(_ context.Context) string {
	return "Ensures that if user_data is not set, all required attributes configured."
}

func (av requiresValidUserData) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	if !request.ConfigValue.IsNull() && !request.ConfigValue.IsUnknown() {
		// skip validation if user_data is defined
		return
	}

	expressions := request.PathExpression.MergeExpressions(av.requiredExpressions...)

	for _, expression := range expressions {
		matchedPaths, diags := request.Config.PathMatches(ctx, expression)
		response.Diagnostics.Append(diags...)
		// collect all errors
		if diags.HasError() {
			continue
		}

		for _, matchedPath := range matchedPaths {
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
			if matchedPathValue.IsNull() {
				response.Diagnostics.AddAttributeError(
					request.Path,
					"Missing required attribute",
					fmt.Sprintf("Attribute %q must be specified when %q is not set.", matchedPath, "user_data"),
				)
			}
		}
	}
}

func RequiresValidUserData(expressions ...path.Expression) validator.String {
	return &requiresValidUserData{
		requiredExpressions: expressions,
	}
}
