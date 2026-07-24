package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Masumoou/terraform-provider-nextcloud/internal/client"
)

var _ resource.Resource = &LDAPConfigResource{}
var _ resource.ResourceWithImportState = &LDAPConfigResource{}

func NewLDAPConfigResource() resource.Resource { return &LDAPConfigResource{} }

type LDAPConfigResource struct {
	client *client.Client
}

type ldapConfigModel struct {
	ConfigID        types.String `tfsdk:"config_id"`
	Host            types.String `tfsdk:"host"`
	Port            types.Int64  `tfsdk:"port"`
	BaseDN          types.String `tfsdk:"base_dn"`
	BindDN          types.String `tfsdk:"bind_dn"`
	BindPassword    types.String `tfsdk:"bind_password"`
	UserFilter      types.String `tfsdk:"user_filter"`
	GroupFilter     types.String `tfsdk:"group_filter"`
	LoginFilter     types.String `tfsdk:"login_filter"`
	UUIDAttribute   types.String `tfsdk:"uuid_attribute"`
	EmailAttribute  types.String `tfsdk:"email_attribute"`
	DisplayNameAttr types.String `tfsdk:"display_name_attribute"`
	NestedGroups    types.Bool   `tfsdk:"nested_groups"`
	CacheTTLSeconds types.Int64  `tfsdk:"cache_ttl_seconds"`
	TLS             types.Bool   `tfsdk:"tls"`
}

func (r *LDAPConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ldap_config"
}

func (r *LDAPConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Nextcloud LDAP backend configuration end-to-end (host, filters, attribute " +
			"mapping, nested groups, caching) without hand-run `occ ldap:set-config` commands.",
		Attributes: map[string]schema.Attribute{
			"config_id": schema.StringAttribute{
				Required:    true,
				Description: "LDAP config prefix used by Nextcloud, e.g. 's01'.",
			},
			"host": schema.StringAttribute{
				Required:    true,
				Description: "LDAP host, e.g. ldaps://ldap.example.com.",
			},
			"port": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "LDAP port (defaults to 636 for TLS, 389 otherwise).",
			},
			"base_dn": schema.StringAttribute{
				Required: true,
			},
			"bind_dn": schema.StringAttribute{
				Required: true,
			},
			"bind_password": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
			},
			"user_filter": schema.StringAttribute{
				Required: true,
			},
			"group_filter": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"login_filter": schema.StringAttribute{
				Required: true,
			},
			"uuid_attribute": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Defaults to 'auto'.",
			},
			"email_attribute": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"display_name_attribute": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"nested_groups": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"cache_ttl_seconds": schema.Int64Attribute{
				Optional: true,
				Computed: true,
			},
			"tls": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
		},
	}
}

func (r *LDAPConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *LDAPConfigResource) toAPI(m ldapConfigModel) client.LDAPConfig {
	return client.LDAPConfig{
		ConfigID:        m.ConfigID.ValueString(),
		Host:            m.Host.ValueString(),
		Port:            m.Port.ValueInt64(),
		BaseDN:          m.BaseDN.ValueString(),
		BindDN:          m.BindDN.ValueString(),
		BindPassword:    m.BindPassword.ValueString(),
		UserFilter:      m.UserFilter.ValueString(),
		GroupFilter:     m.GroupFilter.ValueString(),
		LoginFilter:     m.LoginFilter.ValueString(),
		UUIDAttribute:   m.UUIDAttribute.ValueString(),
		EmailAttribute:  m.EmailAttribute.ValueString(),
		DisplayNameAttr: m.DisplayNameAttr.ValueString(),
		NestedGroups:    m.NestedGroups.ValueBool(),
		CacheTTLSeconds: m.CacheTTLSeconds.ValueInt64(),
		TLS:             m.TLS.ValueBool(),
	}
}

func (r *LDAPConfigResource) fromAPI(cfg *client.LDAPConfig, m *ldapConfigModel) {
	m.ConfigID = types.StringValue(cfg.ConfigID)
	m.Host = types.StringValue(cfg.Host)
	m.Port = types.Int64Value(cfg.Port)
	m.BaseDN = types.StringValue(cfg.BaseDN)
	m.BindDN = types.StringValue(cfg.BindDN)
	// bind_password is write-only from the API's perspective; keep whatever
	// is already in state/plan rather than overwriting with an empty value.
	m.UserFilter = types.StringValue(cfg.UserFilter)
	m.GroupFilter = types.StringValue(cfg.GroupFilter)
	m.LoginFilter = types.StringValue(cfg.LoginFilter)
	m.UUIDAttribute = types.StringValue(cfg.UUIDAttribute)
	m.EmailAttribute = types.StringValue(cfg.EmailAttribute)
	m.DisplayNameAttr = types.StringValue(cfg.DisplayNameAttr)
	m.NestedGroups = types.BoolValue(cfg.NestedGroups)
	m.CacheTTLSeconds = types.Int64Value(cfg.CacheTTLSeconds)
	m.TLS = types.BoolValue(cfg.TLS)
}

func (r *LDAPConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ldapConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.UpsertLDAPConfig(ctx, r.toAPI(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating LDAP config", err.Error())
		return
	}
	pw := plan.BindPassword
	r.fromAPI(out, &plan)
	plan.BindPassword = pw

	if ok, msg, err := r.client.TestLDAPConnection(ctx, plan.ConfigID.ValueString()); err != nil {
		resp.Diagnostics.AddWarning("LDAP connection test failed to run", err.Error())
	} else if !ok {
		resp.Diagnostics.AddWarning("LDAP connection test did not pass", msg)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LDAPConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ldapConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.GetLDAPConfig(ctx, state.ConfigID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading LDAP config", err.Error())
		return
	}
	pw := state.BindPassword
	r.fromAPI(out, &state)
	state.BindPassword = pw
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *LDAPConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ldapConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.UpsertLDAPConfig(ctx, r.toAPI(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating LDAP config", err.Error())
		return
	}
	pw := plan.BindPassword
	r.fromAPI(out, &plan)
	plan.BindPassword = pw
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LDAPConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ldapConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteLDAPConfig(ctx, state.ConfigID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting LDAP config", err.Error())
	}
}

func (r *LDAPConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("config_id"), req, resp)
}
