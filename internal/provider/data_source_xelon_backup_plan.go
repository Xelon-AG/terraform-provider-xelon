package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

var (
	_ datasource.DataSource              = (*backupPlanDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*backupPlanDataSource)(nil)
)

// backupPlanDataSource is the backup plan data source implementation.
type backupPlanDataSource struct {
	client *xelon.Client
}

// backupPlanDataSourceModel maps the backup plan data source schema data.
type backupPlanDataSourceModel struct {
	CloudID types.String `tfsdk:"cloud_id"`
	ID      types.Int64  `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
}

func NewBackupPlanDataSource() datasource.DataSource {
	return &backupPlanDataSource{}
}

func (d *backupPlanDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = "xelon_backup_plan"
}

func (d *backupPlanDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The backup plan data source resolves an existing cloud-owned backup plan by its exact, case-sensitive name.
`,
		Attributes: map[string]schema.Attribute{
			"cloud_id": schema.StringAttribute{
				MarkdownDescription: "ID of the cloud that owns the backup plan.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"id": schema.Int64Attribute{
				MarkdownDescription: "ID of the backup plan.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Exact, case-sensitive name of the backup plan.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

func (d *backupPlanDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	client, ok := request.ProviderData.(*xelon.Client)
	if !ok {
		response.Diagnostics.AddError(
			"Unconfigured Xelon client",
			"Please report this issue to the provider developers.",
		)
		return
	}

	d.client = client
}

func (d *backupPlanDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data backupPlanDataSourceModel

	diags := request.Config.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	cloudID := data.CloudID.ValueString()
	backupPlanName := data.Name.ValueString()

	tflog.Debug(ctx, "reading backup plan", map[string]any{
		"cloud_id": cloudID,
		"name":     backupPlanName,
	})

	tflog.Trace(ctx, "listing backup plans via API", map[string]any{"cloud_id": cloudID})
	backupPlans, _, err := d.client.Backups.ListPlans(ctx, cloudID)
	if err != nil {
		response.Diagnostics.AddError("Unable to read backup plan", err.Error())
		return
	}
	tflog.Trace(ctx, "received backup plans from API", map[string]any{
		"cloud_id": cloudID,
		"count":    len(backupPlans),
	})

	backupPlan, matchCount := findBackupPlanByName(backupPlans, backupPlanName)
	switch matchCount {
	case 0:
		response.Diagnostics.AddError(
			"Backup plan not found",
			fmt.Sprintf("No backup plan named %q was found in cloud %q. Names are matched exactly and case-sensitively.", backupPlanName, cloudID),
		)
		return
	case 1:
		data.ID = types.Int64Value(int64(backupPlan.ID))
		data.Name = types.StringValue(backupPlan.Name)
	default:
		response.Diagnostics.AddError(
			"Ambiguous backup plan",
			fmt.Sprintf("Found %d backup plans named %q in cloud %q. Backup plan names must be unique within a cloud for this data source.", matchCount, backupPlanName, cloudID),
		)
		return
	}

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func findBackupPlanByName(backupPlans []xelon.BackupPlan, name string) (*xelon.BackupPlan, int) {
	var match *xelon.BackupPlan
	matchCount := 0

	for i := range backupPlans {
		if backupPlans[i].Name == name {
			match = &backupPlans[i]
			matchCount++
		}
	}

	return match, matchCount
}
