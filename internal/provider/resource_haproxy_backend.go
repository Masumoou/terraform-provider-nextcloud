package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Masumoou/terraform-provider-nextcloud/internal/client"
)

var _ resource.Resource = &HAProxyBackendResource{}
var _ resource.ResourceWithImportState = &HAProxyBackendResource{}

func NewHAProxyBackendResource() resource.Resource { return &HAProxyBackendResource{} }

type HAProxyBackendResource struct {
	client *client.Client
}

type haproxyServerModel struct {
	Name    types.String `tfsdk:"name"`
	Address types.String `tfsdk:"address"`
	Port    types.Int64  `tfsdk:"port"`
	Check   types.Bool   `tfsdk:"check"`
	Weight  types.Int64  `tfsdk:"weight"`
}

type haproxyBackendModel struct {
	Name    types.String         `tfsdk:"name"`
	Mode    types.String         `tfsdk:"mode"`
	Balance types.String         `tfsdk:"balance"`
	Servers []haproxyServerModel `tfsdk:"server"`
}

func (r *HAProxyBackendResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_haproxy_backend"
}

func (r *HAProxyBackendResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an HAProxy backend (and its servers) via the HAProxy Data Plane API — frontends, " +
			"ACLs, and SSL certs can be added as sibling resources following the same pattern.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},
			"mode": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "'http' or 'tcp'.",
			},
			"balance": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "e.g. 'roundrobin', 'leastconn', 'source'.",
			},
		},
		Blocks: map[string]schema.Block{
			"server": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name":    schema.StringAttribute{Required: true},
						"address": schema.StringAttribute{Required: true},
						"port":    schema.Int64Attribute{Required: true},
						"check":   schema.BoolAttribute{Optional: true, Computed: true},
						"weight":  schema.Int64Attribute{Optional: true, Computed: true},
					},
				},
			},
		},
	}
}

func (r *HAProxyBackendResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *HAProxyBackendResource) toAPI(m haproxyBackendModel) client.HAProxyBackend {
	servers := make(map[string]client.HAProxyServer, len(m.Servers))
	for _, s := range m.Servers {
		checkStr := "disabled"
		if s.Check.ValueBool() {
			checkStr = "enabled"
		}
		servers[s.Name.ValueString()] = client.HAProxyServer{
			Name:    s.Name.ValueString(),
			Address: s.Address.ValueString(),
			Port:    s.Port.ValueInt64(),
			Check:   checkStr,
			Weight:  s.Weight.ValueInt64(),
		}
	}
	var balance *client.HAProxyBalance
	if m.Balance.ValueString() != "" {
		balance = &client.HAProxyBalance{Algorithm: m.Balance.ValueString()}
	}
	return client.HAProxyBackend{
		Name:    m.Name.ValueString(),
		Mode:    m.Mode.ValueString(),
		Balance: balance,
		Servers: servers,
	}
}

func (r *HAProxyBackendResource) fromAPI(b *client.HAProxyBackend, m *haproxyBackendModel) {
	m.Name = types.StringValue(b.Name)
	m.Mode = types.StringValue(b.Mode)
	if b.Balance != nil {
		m.Balance = types.StringValue(b.Balance.Algorithm)
	} else {
		m.Balance = types.StringValue("")
	}
	servers := make([]haproxyServerModel, 0, len(b.Servers))
	for name, s := range b.Servers {
		servers = append(servers, haproxyServerModel{
			Name:    types.StringValue(name),
			Address: types.StringValue(s.Address),
			Port:    types.Int64Value(s.Port),
			Check:   types.BoolValue(s.Check == "enabled"),
			Weight:  types.Int64Value(s.Weight),
		})
	}
	m.Servers = servers
}

func (r *HAProxyBackendResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan haproxyBackendModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.CreateHAProxyBackend(ctx, r.toAPI(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating HAProxy backend", err.Error())
		return
	}
	r.fromAPI(out, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HAProxyBackendResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state haproxyBackendModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.GetHAProxyBackend(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading HAProxy backend", err.Error())
		return
	}
	r.fromAPI(out, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *HAProxyBackendResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan haproxyBackendModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.UpdateHAProxyBackend(ctx, r.toAPI(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating HAProxy backend", err.Error())
		return
	}
	r.fromAPI(out, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HAProxyBackendResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state haproxyBackendModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteHAProxyBackend(ctx, state.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting HAProxy backend", err.Error())
	}
}

func (r *HAProxyBackendResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
