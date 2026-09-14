// Package validators holds the validators specific to SecObserve's API.
//
// Generic ones (OneOf, Between, AlsoRequires) come from
// terraform-plugin-framework-validators; only rules the upstream package
// cannot express live here.
package validators

import (
	"context"
	"net/url"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// RegularExpression reports a value that Python's re module would reject.
//
// SecObserve compiles these server-side (core/api/serializers_product.py:653)
// and answers with a 400. Go's regexp is RE2 and not identical to Python's
// engine, so this catches the common mistakes at plan time without claiming to
// be exhaustive.
func RegularExpression() validator.String {
	return regularExpressionValidator{}
}

type regularExpressionValidator struct{}

func (v regularExpressionValidator) Description(context.Context) string {
	return "must be a valid regular expression"
}

func (v regularExpressionValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v regularExpressionValidator) ValidateString(
	_ context.Context, req validator.StringRequest, resp *validator.StringResponse,
) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if value == "" {
		return
	}
	if _, err := regexp.Compile(value); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid regular expression",
			"SecObserve compiles this value as a regular expression and would reject it: "+err.Error(),
		)
	}
}

// HTTPURL reports a value that is not an http or https URL.
//
// Mirrors the backend's validate_url, which SecObserve applies to
// repository_prefix and the two notification webhooks. An empty string is
// allowed, because that is how those fields are cleared.
func HTTPURL() validator.String {
	return httpURLValidator{}
}

type httpURLValidator struct{}

func (v httpURLValidator) Description(context.Context) string {
	return "must be an http or https URL"
}

func (v httpURLValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v httpURLValidator) ValidateString(
	_ context.Context, req validator.StringRequest, resp *validator.StringResponse,
) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if value == "" {
		return
	}

	parsed, err := url.Parse(value)
	if err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid URL", err.Error())
		return
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid URL",
			"SecObserve only accepts http and https URLs here; got scheme "+parsed.Scheme+".",
		)
	}
}
