package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Masumoou/terraform-provider-nextcloud/internal/client"
)

var _ resource.Resource = &RedisConfigResource{}

func NewRedisConfigResource() resource.Resource { return &RedisConfigResource{} }

type RedisConfigResource struct {
	client *client.Client
}

type redisConfigModel struct {
	ID                types.String `tfsdk:"id"`
	Host              types.String `tfsdk:"host"`
	Port              types.Int64  `tfsdk:"port"`
	Password          types.String `tfsdk:"password"`
	UseForCache       types.Bool   `tfsdk:"use_for_cache"`
	UseForFileLocking types.Bool   `tfsdk:"use_for_file_locking"`
	Timeout           types.Int64  `tfsdk:"timeout_seconds"`
}

func (r *RedisConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_redis_config"
}

func (r *RedisConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the Redis connection Nextcloud uses for caching and/or file locking.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"host": schema.StringAttribute{
				Required: true,
			},
			"port": schema.Int64Attribute{
				Optional: true,
				Computed: true,
			},
			"password": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
			},
			"use_for_cache": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"use_for_file_locking": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"timeout_seconds": schema.Int64Attribute{
				Optional: true,
				Computed: true,
			},
		},
	}
}

func (r *RedisConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *RedisConfigResource) toAPI(m redisConfigModel) client.RedisConfig {
	return client.RedisConfig{
		Host:              m.Host.ValueString(),
		Port:              m.Port.ValueInt64(),
		Password:          m.Password.ValueString(),
		UseForCache:       m.UseForCache.ValueBool(),
		UseForFileLocking: m.UseForFileLocking.ValueBool(),
		Timeout:           m.Timeout.ValueInt64(),
	}
}

func (r *RedisConfigResource) fromAPI(cfg *client.RedisConfig, m *redisConfigModel) {
	m.ID = types.StringValue("redis-config")
	m.Host = types.StringValue(cfg.Host)
	m.Port = types.Int64Value(cfg.Port)
	m.UseForCache = types.BoolValue(cfg.UseForCache)
	m.UseForFileLocking = types.BoolValue(cfg.UseForFileLocking)
	m.Timeout = types.Int64Value(cfg.Timeout)
}

func (r *RedisConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan redisConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.PutRedisConfig(ctx, r.toAPI(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error applying Redis config", err.Error())
		return
	}
	pw := plan.Password
	r.fromAPI(out, &plan)
	plan.Password = pw
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RedisConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state redisConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.GetRedisConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading Redis config", err.Error())
		return
	}
	pw := state.Password
	r.fromAPI(out, &state)
	state.Password = pw
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RedisConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan redisConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.PutRedisConfig(ctx, r.toAPI(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating Redis config", err.Error())
		return
	}
	pw := plan.Password
	r.fromAPI(out, &plan)
	plan.Password = pw
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RedisConfigResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Singleton config: deletion only drops it from Terraform state.
}
