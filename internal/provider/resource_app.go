package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Masumoou/terraform-provider-nextcloud/internal/client"
)

var _ resource.Resource = &AppResource{}
var _ resource.ResourceWithImportState = &AppResource{}

func NewAppResource() resource.Resource { return &AppResource{} }

type AppResource struct {
	client *client.Client
}

type appModel struct {
	AppID   types.String `tfsdk:"app_id"`
	Enabled types.Bool   `tfsdk:"enabled"`
	Version types.String `tfsdk:"version"`
	Config  types.Map    `tfsdk:"config"`
}

func (r *AppResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app"
}

func (r *AppResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Installs/enables/disables a Nextcloud app and manages its app-level config values, " +
			"mirroring `occ app:install`, `occ app:enable/disable`, and `occ config:app:set`.",
		Attributes: map[string]schema.Attribute{
			"app_id": schema.StringAttribute{
				Required:    true,
				Description: "Nextcloud app id, e.g. 'onlyoffice', 'files_external', 'user_ldap'.",
			},
			"enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"version": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Pin to a specific version; omit to track whatever is installed.",
			},
			"config": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "App-level config:app:set key/value pairs.",
			},
		},
	}
}

func (r *AppResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *AppResource) toAPI(ctx context.Context, m appModel) client.App {
	cfg := map[string]string{}
	m.Config.ElementsAs(ctx, &cfg, false)
	return client.App{
		AppID:   m.AppID.ValueString(),
		Enabled: m.Enabled.ValueBool(),
		Version: m.Version.ValueString(),
		Config:  cfg,
	}
}

func (r *AppResource) fromAPI(ctx context.Context, a *client.App, m *appModel) {
	m.AppID = types.StringValue(a.AppID)
	m.Enabled = types.BoolValue(a.Enabled)
	m.Version = types.StringValue(a.Version)
	cfgVal, _ := types.MapValueFrom(ctx, types.StringType, a.Config)
	m.Config = cfgVal
}

func (r *AppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan appModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.UpsertApp(ctx, r.toAPI(ctx, plan))
	if err != nil {
		resp.Diagnostics.AddError("Error installing/configuring app", err.Error())
		return
	}
	r.fromAPI(ctx, out, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state appModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.GetApp(ctx, state.AppID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading app", err.Error())
		return
	}
	r.fromAPI(ctx, out, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan appModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.UpsertApp(ctx, r.toAPI(ctx, plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating app", err.Error())
		return
	}
	r.fromAPI(ctx, out, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state appModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteApp(ctx, state.AppID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error removing app", err.Error())
	}
}

func (r *AppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("app_id"), req, resp)
}
