// Package api_configuration implements the secobserve_api_configuration
// resource.
package api_configuration //nolint:revive // the package name mirrors the Terraform resource name

import (
	"context"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jabbrwcky/terraform-provider-secobserve/internal/client"
	"github.com/jabbrwcky/terraform-provider-secobserve/internal/schemacommon"
	"github.com/jabbrwcky/terraform-provider-secobserve/internal/tfutil"
	sovalidators "github.com/jabbrwcky/terraform-provider-secobserve/internal/validators"
)

var (
	_ resource.ResourceWithConfigure   = (*apiConfigurationResource)(nil)
	_ resource.ResourceWithImportState = (*apiConfigurationResource)(nil)
)

// New returns the secobserve_api_configuration resource.
func New() resource.Resource {
	return &apiConfigurationResource{}
}

type apiConfigurationResource struct {
	client *client.Client
}

type model struct {
	ID      types.Int64  `tfsdk:"id"`
	Product types.Int64  `tfsdk:"product"`
	Name    types.String `tfsdk:"name"`
	Parser  types.Int64  `tfsdk:"parser"`
	BaseURL types.String `tfsdk:"base_url"`

	ProjectKey types.String `tfsdk:"project_key"`
	APIKey     types.String `tfsdk:"api_key"`
	Query      types.String `tfsdk:"query"`

	BasicAuthEnabled  types.Bool   `tfsdk:"basic_auth_enabled"`
	BasicAuthUsername types.String `tfsdk:"basic_auth_username"`
	BasicAuthPassword types.String `tfsdk:"basic_auth_password"`
	VerifySSL         types.Bool   `tfsdk:"verify_ssl"`

	AutomaticImportEnabled            types.Bool   `tfsdk:"automatic_import_enabled"`
	AutomaticImportBranch             types.Int64  `tfsdk:"automatic_import_branch"`
	AutomaticImportService            types.Int64  `tfsdk:"automatic_import_service"`
	AutomaticImportDockerImageNameTag types.String `tfsdk:"automatic_import_docker_image_name_tag"`
	AutomaticImportEndpointURL        types.String `tfsdk:"automatic_import_endpoint_url"`
	AutomaticImportKubernetesCluster  types.String `tfsdk:"automatic_import_kubernetes_cluster"`

	// TestConnection is write-only: it is never returned by the API (it is a
	// side-effecting flag, not stored state) and is never compared against a
	// prior value, so it belongs in state for neither drift detection nor any
	// other purpose.
	TestConnection types.Bool `tfsdk:"test_connection"`
}

func (r *apiConfigurationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_configuration"
}

func (r *apiConfigurationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = tfutil.ResourceClient(req, resp)
}

func (r *apiConfigurationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A per-product configuration for importing observations from a scanner's API " +
			"(as opposed to uploading a scan result file).",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{schemacommon.Int64UseStateForUnknown()},
				MarkdownDescription: "Numeric id of the configuration.",
			},
			"product": schema.Int64Attribute{
				Required:      true,
				Validators:    []validator.Int64{int64validator.AtLeast(1)},
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
				MarkdownDescription: "Id of the product this configuration imports into. Immutable: " +
					"SecObserve rejects moving a configuration to a different product, so a change " +
					"replaces it.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 255)},
				MarkdownDescription: "Name of the configuration. Unique per product.",
			},
			"parser": schema.Int64Attribute{
				Required:   true,
				Validators: []validator.Int64{int64validator.AtLeast(1)},
				MarkdownDescription: "Id of the parser to use. Use the `secobserve_parser` data source, " +
					"filtered to `source = \"API\"`, to resolve a parser name.",
			},
			"base_url": schema.StringAttribute{
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 255)},
				MarkdownDescription: "Base URL of the scanner's API.",
			},
			"project_key": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				Validators:          []validator.String{stringvalidator.LengthAtMost(255)},
				MarkdownDescription: "Project key or identifier within the scanner, if it needs one.",
			},
			"api_key": schema.StringAttribute{
				Optional:  true,
				Computed:  true,
				Sensitive: true,
				Default:   stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.LengthAtMost(255),
				},
				MarkdownDescription: "API key for the scanner. Encrypted at rest; SecObserve strips it from " +
					"responses unless the caller can edit the configuration, which a superuser token " +
					"always can.",
			},
			"query": schema.StringAttribute{
				Optional:   true,
				Computed:   true,
				Default:    stringdefault.StaticString(""),
				Validators: []validator.String{stringvalidator.LengthAtMost(255)},
				MarkdownDescription: "Query used to select results, for scanners that need one (for " +
					"example a Prometheus query for the Trivy Operator).",
			},
			"basic_auth_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether to authenticate with HTTP Basic auth in addition to `api_key`.",
			},
			"basic_auth_username": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				Validators:          []validator.String{stringvalidator.LengthAtMost(255)},
				MarkdownDescription: "HTTP Basic auth username.",
			},
			"basic_auth_password": schema.StringAttribute{
				Optional:   true,
				Computed:   true,
				Sensitive:  true,
				Default:    stringdefault.StaticString(""),
				Validators: []validator.String{stringvalidator.LengthAtMost(255)},
				MarkdownDescription: "HTTP Basic auth password. Encrypted at rest and, unlike `api_key`, " +
					"never stripped from a response.",
			},
			"verify_ssl": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether to verify the scanner's TLS certificate.",
			},
			"automatic_import_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether SecObserve imports from this configuration on its own schedule.",
			},
			"automatic_import_branch": schema.Int64Attribute{
				Optional:   true,
				Validators: []validator.Int64{int64validator.AtLeast(1)},
				MarkdownDescription: "Id of the branch new observations are imported into. Must belong to " +
					"`product`; SecObserve rejects one that does not with a 400.",
			},
			"automatic_import_service": schema.Int64Attribute{
				Optional:            true,
				Validators:          []validator.Int64{int64validator.AtLeast(1)},
				MarkdownDescription: "Id of the service new observations are attributed to.",
			},
			"automatic_import_docker_image_name_tag": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				Validators:          []validator.String{stringvalidator.LengthAtMost(513)},
				MarkdownDescription: "\"<image name>:<tag>\" to associate imported observations with.",
			},
			"automatic_import_endpoint_url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				Validators:          []validator.String{stringvalidator.LengthAtMost(2048), sovalidators.HTTPURL()},
				MarkdownDescription: "Endpoint URL to associate imported observations with.",
			},
			"automatic_import_kubernetes_cluster": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				Validators:          []validator.String{stringvalidator.LengthAtMost(255)},
				MarkdownDescription: "Kubernetes cluster name to associate imported observations with.",
			},
			"test_connection": schema.BoolAttribute{
				Optional:  true,
				WriteOnly: true,
				MarkdownDescription: "When `true`, SecObserve performs a live connection check against the " +
					"scanner using the rest of this configuration as part of the apply, and fails the whole " +
					"request if it does not succeed.\n\n" +
					"~> This is a side-effecting action, not stored state: it never appears in a read and " +
					"cannot cause drift. Set it in an `ephemeral` or pass it via `-var` on the applies where " +
					"a live check is actually wanted, rather than leaving it `true` in committed " +
					"configuration -- every apply would re-run the check.",
			},
		},
	}
}

func (m model) toRequest() client.APIConfigurationRequest {
	return client.APIConfigurationRequest{
		Product:                           m.Product.ValueInt64(),
		Name:                              m.Name.ValueString(),
		Parser:                            m.Parser.ValueInt64(),
		BaseURL:                           m.BaseURL.ValueString(),
		ProjectKey:                        tfutil.StringValue(m.ProjectKey),
		APIKey:                            tfutil.StringValue(m.APIKey),
		Query:                             tfutil.StringValue(m.Query),
		BasicAuthEnabled:                  tfutil.BoolValue(m.BasicAuthEnabled),
		BasicAuthUsername:                 tfutil.StringValue(m.BasicAuthUsername),
		BasicAuthPassword:                 tfutil.StringValue(m.BasicAuthPassword),
		VerifySSL:                         tfutil.BoolValue(m.VerifySSL),
		AutomaticImportEnabled:            tfutil.BoolValue(m.AutomaticImportEnabled),
		AutomaticImportBranch:             tfutil.Int64Ptr(m.AutomaticImportBranch),
		AutomaticImportService:            tfutil.Int64Ptr(m.AutomaticImportService),
		AutomaticImportDockerImageNameTag: tfutil.StringValue(m.AutomaticImportDockerImageNameTag),
		AutomaticImportEndpointURL:        tfutil.StringValue(m.AutomaticImportEndpointURL),
		AutomaticImportKubernetesCluster:  tfutil.StringValue(m.AutomaticImportKubernetesCluster),
		TestConnection:                    tfutil.BoolValue(m.TestConnection),
	}
}

func (m *model) fromAPI(config client.APIConfiguration) {
	m.ID = types.Int64Value(config.ID)
	m.Product = types.Int64Value(config.Product)
	m.Name = types.StringValue(config.Name)
	m.Parser = types.Int64Value(config.Parser)
	m.BaseURL = types.StringValue(config.BaseURL)
	m.ProjectKey = types.StringValue(config.ProjectKey)
	m.APIKey = types.StringValue(config.APIKey)
	m.Query = types.StringValue(config.Query)
	m.BasicAuthEnabled = types.BoolValue(config.BasicAuthEnabled)
	m.BasicAuthUsername = types.StringValue(config.BasicAuthUsername)
	m.BasicAuthPassword = types.StringValue(config.BasicAuthPassword)
	m.VerifySSL = types.BoolValue(config.VerifySSL)
	m.AutomaticImportEnabled = types.BoolValue(config.AutomaticImportEnabled)
	m.AutomaticImportBranch = tfutil.Int64(config.AutomaticImportBranch)
	m.AutomaticImportService = tfutil.Int64(config.AutomaticImportService)
	m.AutomaticImportDockerImageNameTag = types.StringValue(config.AutomaticImportDockerImageNameTag)
	m.AutomaticImportEndpointURL = types.StringValue(config.AutomaticImportEndpointURL)
	m.AutomaticImportKubernetesCluster = types.StringValue(config.AutomaticImportKubernetesCluster)
	// test_connection is write-only and therefore never populated from a
	// response; leave it null so it never appears as a diff.
}

func (r *apiConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	// Write-only attributes are never populated on the plan (Terraform does
	// not store them in plan or state artifacts), so test_connection has to
	// be read from Config, after Plan.Get, or this would overwrite it back
	// to null.
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("test_connection"), &plan.TestConnection)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateAPIConfiguration(ctx, plan.toRequest())
	if err != nil {
		resp.Diagnostics.AddError("Could not create SecObserve API configuration", err.Error())
		return
	}

	var state model
	state.fromAPI(created)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *apiConfigurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := r.client.APIConfiguration(ctx, state.ID.ValueInt64())
	if err != nil {
		if client.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Could not read SecObserve API configuration", err.Error())
		return
	}

	var refreshed model
	refreshed.fromAPI(config)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *apiConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	// Write-only attributes are never populated on the plan (Terraform does
	// not store them in plan or state artifacts), so test_connection has to
	// be read from Config, after Plan.Get, or this would overwrite it back
	// to null.
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("test_connection"), &plan.TestConnection)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateAPIConfiguration(ctx, state.ID.ValueInt64(), plan.toRequest())
	if err != nil {
		resp.Diagnostics.AddError("Could not update SecObserve API configuration", err.Error())
		return
	}

	var refreshed model
	refreshed.fromAPI(updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *apiConfigurationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteAPIConfiguration(ctx, state.ID.ValueInt64()); err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Could not delete SecObserve API configuration", err.Error())
	}
}

// ImportState takes "<product id>/<configuration name>". test_connection is
// never set on import, matching every other apply: it is not stored state.
func (r *apiConfigurationResource) ImportState(
	ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	productID, name, found := strings.Cut(req.ID, "/")
	if !found {
		resp.Diagnostics.AddError(
			"Invalid import id",
			"Import an API configuration as \"<product id>/<configuration name>\", for example \"12/Trivy\".",
		)
		return
	}

	parsedProduct, err := strconv.ParseInt(productID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid import id",
			"The product part of \"<product id>/<configuration name>\" must be numeric: "+err.Error(),
		)
		return
	}

	config, err := r.client.APIConfigurationByName(ctx, parsedProduct, name)
	if err != nil {
		resp.Diagnostics.AddError("Could not import SecObserve API configuration", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), config.ID)...)
}
