package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Masumoou/terraform-provider-nextcloud/internal/client"
)

// Ensure DiskbgProvider satisfies the provider.Provider interface.
var _ provider.Provider = &DiskbgProvider{}

type DiskbgProvider struct {
	version string
}

// diskbgProviderModel maps provider schema data.
type diskbgProviderModel struct {
	AgentURL            types.String `tfsdk:"agent_url"`
	AgentToken          types.String `tfsdk:"agent_token"`
	HAProxyDataPlaneURL types.String `tfsdk:"haproxy_dataplane_url"`
	HAProxyUsername     types.String `tfsdk:"haproxy_username"`
	HAProxyPassword     types.String `tfsdk:"haproxy_password"`
	CephRGWAdminURL     types.String `tfsdk:"ceph_rgw_admin_url"`
	CephRGWAccessKey    types.String `tfsdk:"ceph_rgw_access_key"`
	CephRGWSecretKey    types.String `tfsdk:"ceph_rgw_secret_key"`
	InsecureSkipVerify  types.Bool   `tfsdk:"insecure_skip_verify"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &DiskbgProvider{version: version}
	}
}

func (p *DiskbgProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "nextcloud"
	resp.Version = p.version
}

func (p *DiskbgProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages your Nextcloud platform and supporting infrastructure " +
			"(Nextcloud, LDAP, OnlyOffice, users/apps, occ commands, Redis, HAProxy, Ceph RGW) as code.",
		Attributes: map[string]schema.Attribute{
			"agent_url": schema.StringAttribute{
				Optional: true,
				Description: "Base URL of the diskbg-agent running on the target WFE, e.g. " +
					"https://wfe-01.example.com:8443. Falls back to DISKBG_AGENT_URL.",
			},
			"agent_token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Bearer token for the diskbg-agent. Falls back to DISKBG_AGENT_TOKEN.",
			},
			"haproxy_dataplane_url": schema.StringAttribute{
				Optional:    true,
				Description: "Base URL of the HAProxy Data Plane API, e.g. https://lb-01.example.com:5555/v3. Falls back to DISKBG_HAPROXY_URL.",
			},
			"haproxy_username": schema.StringAttribute{
				Optional:    true,
				Description: "HAProxy Data Plane API basic auth username. Falls back to DISKBG_HAPROXY_USERNAME.",
			},
			"haproxy_password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "HAProxy Data Plane API basic auth password. Falls back to DISKBG_HAPROXY_PASSWORD.",
			},
			"ceph_rgw_admin_url": schema.StringAttribute{
				Optional:    true,
				Description: "Base URL of the Ceph RGW admin ops API. Falls back to DISKBG_CEPH_RGW_URL.",
			},
			"ceph_rgw_access_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Ceph RGW admin access key. Falls back to DISKBG_CEPH_RGW_ACCESS_KEY.",
			},
			"ceph_rgw_secret_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Ceph RGW admin secret key. Falls back to DISKBG_CEPH_RGW_SECRET_KEY.",
			},
			"insecure_skip_verify": schema.BoolAttribute{
				Optional:    true,
				Description: "Skip TLS certificate verification. Only for staging/dev environments with self-signed certs.",
			},
		},
	}
}

func (p *DiskbgProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfgModel diskbgProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfgModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg := client.Config{
		AgentURL:            firstNonEmpty(cfgModel.AgentURL.ValueString(), os.Getenv("DISKBG_AGENT_URL")),
		AgentToken:          firstNonEmpty(cfgModel.AgentToken.ValueString(), os.Getenv("DISKBG_AGENT_TOKEN")),
		HAProxyDataPlaneURL: firstNonEmpty(cfgModel.HAProxyDataPlaneURL.ValueString(), os.Getenv("DISKBG_HAPROXY_URL")),
		HAProxyUsername:     firstNonEmpty(cfgModel.HAProxyUsername.ValueString(), os.Getenv("DISKBG_HAPROXY_USERNAME")),
		HAProxyPassword:     firstNonEmpty(cfgModel.HAProxyPassword.ValueString(), os.Getenv("DISKBG_HAPROXY_PASSWORD")),
		CephRGWAdminURL:     firstNonEmpty(cfgModel.CephRGWAdminURL.ValueString(), os.Getenv("DISKBG_CEPH_RGW_URL")),
		CephRGWAccessKey:    firstNonEmpty(cfgModel.CephRGWAccessKey.ValueString(), os.Getenv("DISKBG_CEPH_RGW_ACCESS_KEY")),
		CephRGWSecretKey:    firstNonEmpty(cfgModel.CephRGWSecretKey.ValueString(), os.Getenv("DISKBG_CEPH_RGW_SECRET_KEY")),
		InsecureSkipVerify:  cfgModel.InsecureSkipVerify.ValueBool(),
	}

	if cfg.AgentURL == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("agent_url"),
			"Missing diskbg-agent URL",
			"Set agent_url in the provider block or the DISKBG_AGENT_URL environment variable. "+
				"This should point at the diskbg-agent instance for the WFE you intend to manage.",
		)
		return
	}

	c := client.New(cfg)
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *DiskbgProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewNextcloudConfigResource,
		NewLDAPConfigResource,
		NewOnlyOfficeResource,
		NewUserResource,
		NewAppResource,
		NewOccCommandResource,
		NewRedisConfigResource,
		NewHAProxyBackendResource,
		NewCephRGWUserResource,
	}
}

func (p *DiskbgProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewHealthDataSource,
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
