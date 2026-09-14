// Package provider wires the SecObserve API client into terraform-plugin-framework.
package provider

import (
	"context"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/jabbrwcky/terraform-provider-secobserve/internal/client"
	authorizationgroupdatasource "github.com/jabbrwcky/terraform-provider-secobserve/internal/datasource/authorization_group"
	branchdatasource "github.com/jabbrwcky/terraform-provider-secobserve/internal/datasource/branch"
	parserdatasource "github.com/jabbrwcky/terraform-provider-secobserve/internal/datasource/parser"
	periodictasksdatasource "github.com/jabbrwcky/terraform-provider-secobserve/internal/datasource/periodic_tasks"
	productdatasource "github.com/jabbrwcky/terraform-provider-secobserve/internal/datasource/product"
	productgroupdatasource "github.com/jabbrwcky/terraform-provider-secobserve/internal/datasource/product_group"
	servicedatasource "github.com/jabbrwcky/terraform-provider-secobserve/internal/datasource/service"
	userdatasource "github.com/jabbrwcky/terraform-provider-secobserve/internal/datasource/user"
	apiconfigurationresource "github.com/jabbrwcky/terraform-provider-secobserve/internal/resource/api_configuration"
	authorizationgroupresource "github.com/jabbrwcky/terraform-provider-secobserve/internal/resource/authorization_group"
	authorizationgroupmemberresource "github.com/jabbrwcky/terraform-provider-secobserve/internal/resource/authorization_group_member"
	branchresource "github.com/jabbrwcky/terraform-provider-secobserve/internal/resource/branch"
	generalruleresource "github.com/jabbrwcky/terraform-provider-secobserve/internal/resource/general_rule"
	productresource "github.com/jabbrwcky/terraform-provider-secobserve/internal/resource/product"
	productapitokenresource "github.com/jabbrwcky/terraform-provider-secobserve/internal/resource/product_api_token"
	productauthorizationgroupmemberresource "github.com/jabbrwcky/terraform-provider-secobserve/internal/resource/product_authorization_group_member"
	productgroupresource "github.com/jabbrwcky/terraform-provider-secobserve/internal/resource/product_group"
	productmemberresource "github.com/jabbrwcky/terraform-provider-secobserve/internal/resource/product_member"
	productruleresource "github.com/jabbrwcky/terraform-provider-secobserve/internal/resource/product_rule"
	serviceresource "github.com/jabbrwcky/terraform-provider-secobserve/internal/resource/service"
	settingsresource "github.com/jabbrwcky/terraform-provider-secobserve/internal/resource/settings"
	userresource "github.com/jabbrwcky/terraform-provider-secobserve/internal/resource/user"
)

const (
	envBaseURL  = "SECOBSERVE_BASE_URL"
	envAPIToken = "SECOBSERVE_API_TOKEN"
	envInsecure = "SECOBSERVE_INSECURE"
)

// New returns the provider factory for the given build version.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &secObserveProvider{version: version}
	}
}

type secObserveProvider struct {
	version string
}

type providerModel struct {
	BaseURL  types.String `tfsdk:"base_url"`
	APIToken types.String `tfsdk:"api_token"`
	Insecure types.Bool   `tfsdk:"insecure"`
	Timeout  types.String `tfsdk:"timeout"`
}

func (p *secObserveProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "secobserve"
	resp.Version = p.version
}

func (p *secObserveProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages configuration of a [SecObserve](https://github.com/SecObserve/SecObserve) instance: " +
			"products, product groups, branches, services, memberships, rules, API import configurations and license policies.",
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "Base URL of the SecObserve **backend**, for example `https://secobserve-backend.example.com`. " +
					"Note this is the backend, not the frontend URL. May also be set with the `" + envBaseURL + "` environment variable.",
			},
			"api_token": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				MarkdownDescription: "User API token of a **superuser**. May also be set with the `" + envAPIToken + "` environment variable.\n\n" +
					"Create one with:\n\n" +
					"```console\n" +
					"curl -X POST \"$SECOBSERVE_BASE_URL/api/authentication/create_user_api_token/\" \\\n" +
					"  -H 'Content-Type: application/json' \\\n" +
					"  -d '{\"username\":\"admin\",\"password\":\"...\",\"name\":\"terraform\"}'\n" +
					"```\n\n" +
					"A superuser token is required: non-superusers receive filtered list responses and cannot write " +
					"users, general rules, settings or periodic tasks, which shows up as permanent unexplained drift.",
			},
			"insecure": schema.BoolAttribute{
				Optional: true,
				MarkdownDescription: "Skip TLS certificate verification. Intended for local development against a self-signed " +
					"certificate; never enable it against a production instance. May also be set with the `" + envInsecure + "` environment variable.",
			},
			"timeout": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "Per-request timeout as a Go duration, for example `60s`. Defaults to `30s`. " +
					"Raise it if the instance is slow to answer large product lists.",
			},
		},
	}
}

func (p *secObserveProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	baseURL := stringOrEnv(config.BaseURL, envBaseURL)
	if baseURL == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("base_url"),
			"Missing SecObserve base URL",
			"Set the provider's base_url attribute or the "+envBaseURL+" environment variable.",
		)
	}

	apiToken := stringOrEnv(config.APIToken, envAPIToken)
	if apiToken == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token"),
			"Missing SecObserve API token",
			"Set the provider's api_token attribute or the "+envAPIToken+" environment variable.",
		)
	}

	insecure := config.Insecure.ValueBool()
	if config.Insecure.IsNull() {
		insecure = os.Getenv(envInsecure) == "true"
	}

	var timeout time.Duration
	if !config.Timeout.IsNull() && !config.Timeout.IsUnknown() {
		parsed, err := time.ParseDuration(config.Timeout.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("timeout"),
				"Invalid timeout",
				"timeout must be a Go duration such as \"30s\" or \"2m\": "+err.Error(),
			)
		} else {
			timeout = parsed
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	apiClient, err := client.New(client.Config{
		BaseURL:   baseURL,
		APIToken:  apiToken,
		Insecure:  insecure,
		Timeout:   timeout,
		UserAgent: "terraform-provider-secobserve/" + p.version,
	})
	if err != nil {
		resp.Diagnostics.AddError("Invalid SecObserve provider configuration", err.Error())
		return
	}

	verifyInstance(ctx, apiClient, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.DataSourceData = apiClient
	resp.ResourceData = apiClient
	resp.EphemeralResourceData = apiClient
}

// verifyInstance fails fast on an unusable token and warns about conditions
// that produce confusing drift later on.
func verifyInstance(ctx context.Context, apiClient *client.Client, diags *diag.Diagnostics) {
	user, err := apiClient.Me(ctx)
	if err != nil {
		if client.Forbidden(err) || client.NotFound(err) {
			diags.AddError(
				"Could not authenticate against SecObserve",
				"GET /api/users/me/ was rejected. Check that api_token is a valid, unexpired user API token "+
					"and that base_url points at the SecObserve backend.\n\nUnderlying error: "+err.Error(),
			)
			return
		}
		diags.AddError("Could not reach SecObserve", err.Error())
		return
	}

	tflog.Debug(ctx, "authenticated against SecObserve", map[string]any{
		"username":     user.Username,
		"is_superuser": user.IsSuperuser,
	})

	if !user.IsSuperuser {
		diags.AddWarning(
			"SecObserve API token does not belong to a superuser",
			"The token belongs to "+user.Username+", which is not a superuser. SecObserve filters list responses "+
				"for non-superusers and rejects writes to users, general rules, settings and periodic tasks. "+
				"Resources will appear to drift on every plan. Use a superuser token.",
		)
	}

	version, err := apiClient.Version(ctx)
	if err != nil {
		// Not fatal: the provider works, we just cannot verify compatibility.
		tflog.Debug(ctx, "could not read SecObserve version", map[string]any{"error": err.Error()})
		return
	}
	if version.Version == "" || version.Version == client.UnknownVersion {
		// An instance running from source rather than a released image.
		return
	}
	if client.MajorMinor(version.Version) != client.MajorMinor(client.SchemaVersion) {
		diags.AddWarning(
			"SecObserve version differs from the version this provider was built against",
			"The instance reports "+version.Version+"; this provider was generated from the SecObserve "+
				client.SchemaVersion+" API schema. Attributes added or removed between these versions may be "+
				"missing or rejected.",
		)
	}
}

func (p *secObserveProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		productgroupresource.New,
		productresource.New,
		branchresource.New,
		serviceresource.New,
		productmemberresource.New,
		productauthorizationgroupmemberresource.New,
		productapitokenresource.New,
		userresource.New,
		authorizationgroupresource.New,
		authorizationgroupmemberresource.New,
		settingsresource.New,
		generalruleresource.New,
		productruleresource.New,
		apiconfigurationresource.New,
	}
}

func (p *secObserveProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		productdatasource.New,
		productgroupdatasource.New,
		branchdatasource.New,
		servicedatasource.New,
		parserdatasource.NewParserDataSource,
		parserdatasource.NewParsersDataSource,
		userdatasource.New,
		authorizationgroupdatasource.New,
		periodictasksdatasource.New,
	}
}

func (p *secObserveProvider) Functions(_ context.Context) []func() function.Function {
	return nil
}

func stringOrEnv(value types.String, envVar string) string {
	if !value.IsNull() && !value.IsUnknown() {
		return value.ValueString()
	}
	return os.Getenv(envVar)
}
