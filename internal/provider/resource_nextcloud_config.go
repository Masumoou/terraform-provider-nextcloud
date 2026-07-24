package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Masumoou/terraform-provider-nextcloud/internal/client"
)

var _ resource.Resource = &NextcloudConfigResource{}
var _ resource.ResourceWithImportState = &NextcloudConfigResource{}

func NewNextcloudConfigResource() resource.Resource { return &NextcloudConfigResource{} }

type NextcloudConfigResource struct {
	client *client.Client
}

type nextcloudConfigModel struct {
	ID              types.String `tfsdk:"id"`
	MaintenanceMode types.Bool   `tfsdk:"maintenance_mode"`
	TrustedDomains  types.List   `tfsdk:"trusted_domains"`
	TrustedProxies  types.List   `tfsdk:"trusted_proxies"`
	DefaultLanguage types.String `tfsdk:"default_language"`
	DefaultQuota    types.String `tfsdk:"default_quota"`
	UploadLimitMB   types.Int64  `tfsdk:"upload_limit_mb"`
	LogLevel        types.Int64  `tfsdk:"log_level"`
	BackgroundJobs  types.String `tfsdk:"background_jobs"`
	TrashRetention  types.String `tfsdk:"trash_retention_days"`
	VersionsRetain  types.String `tfsdk:"versions_retention_days"`
}

func (r *NextcloudConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config"
}

func (r *NextcloudConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the singleton Nextcloud config.php-level settings for a WFE (maintenance mode, " +
			"trusted domains/proxies, default language/quota, upload limit, logging, background jobs, retention).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Fixed identifier; one config resource manages the whole node.",
			},
			"maintenance_mode": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether Nextcloud is in maintenance mode.",
			},
			"trusted_domains": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "List of trusted_domains entries.",
			},
			"trusted_proxies": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "List of trusted_proxies entries.",
			},
			"default_language": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Default instance language, e.g. 'bg' or 'en'.",
			},
			"default_quota": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Default user quota, e.g. '10 GB' or 'none'.",
			},
			"upload_limit_mb": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Max upload size in MB (maps to PHP upload_max_filesize/post_max_size).",
			},
			"log_level": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Nextcloud loglevel (0=debug .. 4=fatal).",
			},
			"background_jobs": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "One of 'ajax', 'webcron', or 'cron'.",
			},
			"trash_retention_days": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Trashbin retention policy string, e.g. 'auto, 30'.",
			},
			"versions_retention_days": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Versions retention policy string, e.g. 'auto, 90'.",
			},
		},
	}
}

func (r *NextcloudConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *NextcloudConfigResource) toAPI(ctx context.Context, m nextcloudConfigModel) client.NextcloudConfig {
	var domains, proxies []string
	m.TrustedDomains.ElementsAs(ctx, &domains, false)
	m.TrustedProxies.ElementsAs(ctx, &proxies, false)
	return client.NextcloudConfig{
		MaintenanceMode: m.MaintenanceMode.ValueBool(),
		TrustedDomains:  domains,
		TrustedProxies:  proxies,
		DefaultLanguage: m.DefaultLanguage.ValueString(),
		DefaultQuota:    m.DefaultQuota.ValueString(),
		UploadLimitMB:   m.UploadLimitMB.ValueInt64(),
		LogLevel:        m.LogLevel.ValueInt64(),
		BackgroundJobs:  m.BackgroundJobs.ValueString(),
		TrashRetention:  m.TrashRetention.ValueString(),
		VersionsRetain:  m.VersionsRetain.ValueString(),
	}
}

func (r *NextcloudConfigResource) fromAPI(cfg *client.NextcloudConfig, m *nextcloudConfigModel) {
	m.ID = types.StringValue("nextcloud-config")
	m.MaintenanceMode = types.BoolValue(cfg.MaintenanceMode)
	m.TrustedDomains = stringListValue(cfg.TrustedDomains)
	m.TrustedProxies = stringListValue(cfg.TrustedProxies)
	m.DefaultLanguage = types.StringValue(cfg.DefaultLanguage)
	m.DefaultQuota = types.StringValue(cfg.DefaultQuota)
	m.UploadLimitMB = types.Int64Value(cfg.UploadLimitMB)
	m.LogLevel = types.Int64Value(cfg.LogLevel)
	m.BackgroundJobs = types.StringValue(cfg.BackgroundJobs)
	m.TrashRetention = types.StringValue(cfg.TrashRetention)
	m.VersionsRetain = types.StringValue(cfg.VersionsRetain)
}

func (r *NextcloudConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan nextcloudConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.PutNextcloudConfig(ctx, r.toAPI(ctx, plan))
	if err != nil {
		resp.Diagnostics.AddError("Error applying Nextcloud config", err.Error())
		return
	}
	r.fromAPI(out, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *NextcloudConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state nextcloudConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.GetNextcloudConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading Nextcloud config", err.Error())
		return
	}
	r.fromAPI(out, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *NextcloudConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan nextcloudConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.PutNextcloudConfig(ctx, r.toAPI(ctx, plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating Nextcloud config", err.Error())
		return
	}
	r.fromAPI(out, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *NextcloudConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// There is nothing meaningful to "delete" for a singleton config
	// resource; removing it from state simply stops managing the node.
}

func (r *NextcloudConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
