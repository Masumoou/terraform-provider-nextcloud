package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Masumoou/terraform-provider-nextcloud/internal/client"
)

var _ resource.Resource = &OnlyOfficeResource{}

func NewOnlyOfficeResource() resource.Resource { return &OnlyOfficeResource{} }

type OnlyOfficeResource struct {
	client *client.Client
}

type onlyOfficeModel struct {
	ID                types.String `tfsdk:"id"`
	DocumentServerURL types.String `tfsdk:"document_server"`
	InternalURL       types.String `tfsdk:"internal_url"`
	StorageURL        types.String `tfsdk:"storage_url"`
	JWTSecret         types.String `tfsdk:"jwt_secret"`
	JWTHeader         types.String `tfsdk:"jwt_header"`
	JWTEnabled        types.Bool   `tfsdk:"jwt_enabled"`
	VerifySSL         types.Bool   `tfsdk:"verify_ssl"`
	ConnectionTimeout types.Int64  `tfsdk:"connection_timeout"`
	ValidateOnApply   types.Bool   `tfsdk:"validate_on_apply"`
}

func (r *OnlyOfficeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_onlyoffice"
}

func (r *OnlyOfficeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the OnlyOffice Document Server integration for your Nextcloud instance, matching " +
			"`resource \"nextcloud_onlyoffice\" \"main\"` from the platform provider design doc.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"document_server": schema.StringAttribute{
				Required:    true,
				Description: "Public/browser-facing OnlyOffice Document Server URL, e.g. https://office.example.com.",
			},
			"internal_url": schema.StringAttribute{
				Optional:    true,
				Description: "Internal URL used for server-to-server calls, if different from document_server.",
			},
			"storage_url": schema.StringAttribute{
				Optional:    true,
				Description: "URL the Document Server uses to fetch/save files back to Nextcloud, if different from the default.",
			},
			"jwt_secret": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
			},
			"jwt_header": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"jwt_enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"verify_ssl": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"connection_timeout": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Connection timeout in seconds.",
			},
			"validate_on_apply": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Description: "When true (default), runs the end-to-end validation (reachability, JWT match, " +
					"Nextcloud<->Document Server round trip, test document open) after every create/update and " +
					"surfaces failures as a warning.",
			},
		},
	}
}

func (r *OnlyOfficeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *OnlyOfficeResource) toAPI(m onlyOfficeModel) client.OnlyOfficeConfig {
	return client.OnlyOfficeConfig{
		DocumentServerURL: m.DocumentServerURL.ValueString(),
		InternalURL:       m.InternalURL.ValueString(),
		StorageURL:        m.StorageURL.ValueString(),
		JWTSecret:         m.JWTSecret.ValueString(),
		JWTHeader:         m.JWTHeader.ValueString(),
		JWTEnabled:        m.JWTEnabled.ValueBool(),
		VerifySSL:         m.VerifySSL.ValueBool(),
		ConnectionTimeout: m.ConnectionTimeout.ValueInt64(),
	}
}

func (r *OnlyOfficeResource) fromAPI(cfg *client.OnlyOfficeConfig, m *onlyOfficeModel) {
	m.ID = types.StringValue("onlyoffice")
	m.DocumentServerURL = types.StringValue(cfg.DocumentServerURL)
	m.InternalURL = types.StringValue(cfg.InternalURL)
	m.StorageURL = types.StringValue(cfg.StorageURL)
	m.JWTHeader = types.StringValue(cfg.JWTHeader)
	m.JWTEnabled = types.BoolValue(cfg.JWTEnabled)
	m.VerifySSL = types.BoolValue(cfg.VerifySSL)
	m.ConnectionTimeout = types.Int64Value(cfg.ConnectionTimeout)
}

func (r *OnlyOfficeResource) applyAndValidate(ctx context.Context, plan *onlyOfficeModel, diags *diagAppender) {
	secret := plan.JWTSecret
	out, err := r.client.PutOnlyOfficeConfig(ctx, r.toAPI(*plan))
	if err != nil {
		diags.AddError("Error applying OnlyOffice config", err.Error())
		return
	}
	r.fromAPI(out, plan)
	plan.JWTSecret = secret

	if plan.ValidateOnApply.IsNull() || plan.ValidateOnApply.ValueBool() {
		ok, checks, verr := r.client.ValidateOnlyOffice(ctx)
		if verr != nil {
			diags.AddWarning("OnlyOffice validation failed to run", verr.Error())
		} else if !ok {
			diags.AddWarning("OnlyOffice end-to-end validation did not fully pass",
				"Checks performed: "+joinStrings(checks))
		}
	}
}

func (r *OnlyOfficeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan onlyOfficeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.applyAndValidate(ctx, &plan, &diagAppender{&resp.Diagnostics})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OnlyOfficeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state onlyOfficeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.GetOnlyOfficeConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading OnlyOffice config", err.Error())
		return
	}
	secret := state.JWTSecret
	r.fromAPI(out, &state)
	state.JWTSecret = secret
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OnlyOfficeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan onlyOfficeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.applyAndValidate(ctx, &plan, &diagAppender{&resp.Diagnostics})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OnlyOfficeResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Singleton config: deletion only drops it from Terraform state.
}
