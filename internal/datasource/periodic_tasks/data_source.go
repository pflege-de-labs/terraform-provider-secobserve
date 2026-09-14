// Package periodic_tasks implements the secobserve_periodic_tasks data
// source.
package periodic_tasks //nolint:revive // the package name mirrors the Terraform data source name

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jabbrwcky/terraform-provider-secobserve/internal/client"
	"github.com/jabbrwcky/terraform-provider-secobserve/internal/tfutil"
)

var _ datasource.DataSourceWithConfigure = (*periodicTasksDataSource)(nil)

// New returns the secobserve_periodic_tasks data source.
func New() datasource.DataSource {
	return &periodicTasksDataSource{}
}

type periodicTasksDataSource struct {
	client *client.Client
}

type taskModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Task      types.String `tfsdk:"task"`
	StartTime types.String `tfsdk:"start_time"`
	Duration  types.Int64  `tfsdk:"duration"`
	Status    types.String `tfsdk:"status"`
	Message   types.String `tfsdk:"message"`
}

type model struct {
	Task   types.String `tfsdk:"task"`
	Status types.String `tfsdk:"status"`
	Tasks  []taskModel  `tfsdk:"tasks"`
}

func (d *periodicTasksDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_periodic_tasks"
}

func (d *periodicTasksDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = tfutil.DataSourceClient(req, resp)
}

func (d *periodicTasksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists recorded runs of SecObserve's scheduled background jobs, most recent " +
			"first. Read-only: SecObserve writes these itself as jobs run, and scheduling is controlled " +
			"entirely through `secobserve_settings`' `*_crontab_minute`/`*_crontab_hour` attributes, not " +
			"through this data source.",
		Attributes: map[string]schema.Attribute{
			"task": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Only return runs whose task name contains this (case-sensitive substring match).",
			},
			"status": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Only return runs with this status: `Running`, `Success` or `Failure`.",
			},
			"tasks": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Matching runs, most recent first.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":         schema.Int64Attribute{Computed: true, MarkdownDescription: "Numeric id of the run."},
						"task":       schema.StringAttribute{Computed: true, MarkdownDescription: "Name of the task."},
						"start_time": schema.StringAttribute{Computed: true, MarkdownDescription: "When the run started."},
						"duration": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "Duration in milliseconds, null while still running.",
						},
						"status":  schema.StringAttribute{Computed: true, MarkdownDescription: "`Running`, `Success` or `Failure`."},
						"message": schema.StringAttribute{Computed: true, MarkdownDescription: "Error message, if any."},
					},
				},
			},
		},
	}
}

func (d *periodicTasksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := d.client.PeriodicTasks(ctx, config.Task.ValueString(), config.Status.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Could not list SecObserve periodic tasks", err.Error())
		return
	}

	config.Tasks = make([]taskModel, 0, len(found))
	for _, task := range found {
		config.Tasks = append(config.Tasks, taskModel{
			ID:        types.Int64Value(task.ID),
			Task:      types.StringValue(task.Task),
			StartTime: types.StringValue(task.StartTime),
			Duration:  tfutil.Int64(task.Duration),
			Status:    types.StringValue(task.Status),
			Message:   types.StringValue(task.Message),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, config)...)
}
