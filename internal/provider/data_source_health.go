package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Masumoou/terraform-provider-nextcloud/internal/client"
)

var _ datasource.DataSource = &HealthDataSource{}

func NewHealthDataSource() datasource.DataSource { return &HealthDataSource{} }

type HealthDataSource struct {
	client *client.Client
}

type healthModel struct {
	ID         types.String `tfsdk:"id"`
	Apache     types.Bool   `tfsdk:"apache"`
	PHP        types.Bool   `tfsdk:"php"`
	Nextcloud  types.Bool   `tfsdk:"nextcloud"`
	LDAP       types.Bool   `tfsdk:"ldap"`
	PostgreSQL types.Bool   `tfsdk:"postgresql"`
	Redis      types.Bool   `tfsdk:"redis"`
	CephRGW    types.Bool   `tfsdk:"ceph_rgw"`
	OnlyOffice types.Bool   `tfsdk:"onlyoffice"`
	Healthy    types.Bool   `tfsdk:"healthy"`
}

func (d *HealthDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_health"
}

func (d *HealthDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads platform-wide health for a WFE: Apache, PHP, Nextcloud, LDAP, PostgreSQL, Redis, " +
			"Ceph RGW, OnlyOffice, HAProxy. Useful as a post-deploy gate, e.g. with a precondition block.",
		Attributes: map[string]schema.Attribute{
			"id":         schema.StringAttribute{Computed: true},
			"apache":     schema.BoolAttribute{Computed: true},
			"php":        schema.BoolAttribute{Computed: true},
			"nextcloud":  schema.BoolAttribute{Computed: true},
			"ldap":       schema.BoolAttribute{Computed: true},
			"postgresql": schema.BoolAttribute{Computed: true},
			"redis":      schema.BoolAttribute{Computed: true},
			"ceph_rgw":   schema.BoolAttribute{Computed: true},
			"onlyoffice": schema.BoolAttribute{Computed: true},
			"healthy": schema.BoolAttribute{
				Computed:    true,
				Description: "true only if every individual check above is true.",
			},
		},
	}
}

func (d *HealthDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (d *HealthDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	h, err := d.client.GetHealth(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading platform health", err.Error())
		return
	}

	m := healthModel{
		ID:         types.StringValue("health"),
		Apache:     types.BoolValue(h.Apache),
		PHP:        types.BoolValue(h.PHP),
		Nextcloud:  types.BoolValue(h.Nextcloud),
		LDAP:       types.BoolValue(h.LDAP),
		PostgreSQL: types.BoolValue(h.PostgreSQL),
		Redis:      types.BoolValue(h.Redis),
		CephRGW:    types.BoolValue(h.CephRGW),
		OnlyOffice: types.BoolValue(h.OnlyOffice),
		Healthy: types.BoolValue(h.Apache && h.PHP && h.Nextcloud && h.LDAP &&
			h.PostgreSQL && h.Redis && h.CephRGW && h.OnlyOffice),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
