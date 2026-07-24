package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Masumoou/terraform-provider-nextcloud/internal/client"
)

var _ resource.Resource = &CephRGWUserResource{}
var _ resource.ResourceWithImportState = &CephRGWUserResource{}

func NewCephRGWUserResource() resource.Resource { return &CephRGWUserResource{} }

type CephRGWUserResource struct {
	client *client.Client
}

type cephRGWUserModel struct {
	UserID      types.String `tfsdk:"user_id"`
	DisplayName types.String `tfsdk:"display_name"`
	Email       types.String `tfsdk:"email"`
	AccessKey   types.String `tfsdk:"access_key"`
	SecretKey   types.String `tfsdk:"secret_key"`
	MaxBuckets  types.Int64  `tfsdk:"max_buckets"`
	Suspended   types.Bool   `tfsdk:"suspended"`
}

func (r *CephRGWUserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ceph_rgw_user"
}

func (r *CephRGWUserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Ceph RGW (S3-compatible) user via the RGW admin ops API — access/secret keys, " +
			"bucket quota, and suspension state.",
		Attributes: map[string]schema.Attribute{
			"user_id": schema.StringAttribute{
				Required: true,
			},
			"display_name": schema.StringAttribute{
				Required: true,
			},
			"email": schema.StringAttribute{
				Optional: true,
			},
			"access_key": schema.StringAttribute{
				Computed: true,
			},
			"secret_key": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
			},
			"max_buckets": schema.Int64Attribute{
				Optional: true,
				Computed: true,
			},
			"suspended": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
		},
	}
}

func (r *CephRGWUserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *CephRGWUserResource) fromAPI(u *client.CephRGWUser, m *cephRGWUserModel) {
	m.UserID = types.StringValue(u.UserID)
	m.DisplayName = types.StringValue(u.DisplayName)
	m.Email = types.StringValue(u.Email)
	m.AccessKey = types.StringValue(u.AccessKey)
	m.SecretKey = types.StringValue(u.SecretKey)
	m.MaxBuckets = types.Int64Value(u.MaxBuckets)
	m.Suspended = types.BoolValue(u.Suspended)
}

func (r *CephRGWUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan cephRGWUserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.CreateCephRGWUser(ctx, client.CephRGWUser{
		UserID:      plan.UserID.ValueString(),
		DisplayName: plan.DisplayName.ValueString(),
		Email:       plan.Email.ValueString(),
		MaxBuckets:  plan.MaxBuckets.ValueInt64(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating Ceph RGW user", err.Error())
		return
	}
	r.fromAPI(out, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CephRGWUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state cephRGWUserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.GetCephRGWUser(ctx, state.UserID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading Ceph RGW user", err.Error())
		return
	}
	r.fromAPI(out, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update recreates the user against the admin ops API, which only exposes
// PUT-as-upsert semantics for the fields this resource manages.
func (r *CephRGWUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan cephRGWUserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.CreateCephRGWUser(ctx, client.CephRGWUser{
		UserID:      plan.UserID.ValueString(),
		DisplayName: plan.DisplayName.ValueString(),
		Email:       plan.Email.ValueString(),
		MaxBuckets:  plan.MaxBuckets.ValueInt64(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating Ceph RGW user", err.Error())
		return
	}
	r.fromAPI(out, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CephRGWUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state cephRGWUserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteCephRGWUser(ctx, state.UserID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting Ceph RGW user", err.Error())
	}
}

func (r *CephRGWUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("user_id"), req, resp)
}
