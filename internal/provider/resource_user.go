package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Masumoou/terraform-provider-nextcloud/internal/client"
)

var _ resource.Resource = &UserResource{}
var _ resource.ResourceWithImportState = &UserResource{}

func NewUserResource() resource.Resource { return &UserResource{} }

type UserResource struct {
	client *client.Client
}

type userModel struct {
	UserID      types.String `tfsdk:"user_id"`
	DisplayName types.String `tfsdk:"display_name"`
	Email       types.String `tfsdk:"email"`
	Password    types.String `tfsdk:"password"`
	Quota       types.String `tfsdk:"quota"`
	Groups      types.List   `tfsdk:"groups"`
	Enabled     types.Bool   `tfsdk:"enabled"`
}

func (r *UserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *UserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Nextcloud user via the Provisioning API (users, groups, quota, enable/disable).",
		Attributes: map[string]schema.Attribute{
			"user_id": schema.StringAttribute{
				Required: true,
			},
			"display_name": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"email": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Only used on create, or to force a reset. Not readable back from the API.",
			},
			"quota": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "e.g. '10 GB', 'none', 'default'.",
			},
			"groups": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
		},
	}
}

func (r *UserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *UserResource) fromAPI(ctx context.Context, u *client.User, m *userModel) {
	m.UserID = types.StringValue(u.UserID)
	m.DisplayName = types.StringValue(u.DisplayName)
	m.Email = types.StringValue(u.Email)
	m.Quota = types.StringValue(u.QuotaBytes)
	m.Groups = stringListValue(u.Groups)
	m.Enabled = types.BoolValue(u.Enabled)
}

func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var groups []string
	plan.Groups.ElementsAs(ctx, &groups, false)

	out, err := r.client.CreateUser(ctx, client.User{
		UserID:      plan.UserID.ValueString(),
		DisplayName: plan.DisplayName.ValueString(),
		Email:       plan.Email.ValueString(),
		Password:    plan.Password.ValueString(),
		QuotaBytes:  plan.Quota.ValueString(),
		Groups:      groups,
		Enabled:     true,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating user", err.Error())
		return
	}
	pw := plan.Password
	r.fromAPI(ctx, out, &plan)
	plan.Password = pw
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.GetUser(ctx, state.UserID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading user", err.Error())
		return
	}
	pw := state.Password
	r.fromAPI(ctx, out, &state)
	state.Password = pw
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan userModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var groups []string
	plan.Groups.ElementsAs(ctx, &groups, false)

	out, err := r.client.UpdateUser(ctx, client.User{
		UserID:      plan.UserID.ValueString(),
		DisplayName: plan.DisplayName.ValueString(),
		Email:       plan.Email.ValueString(),
		QuotaBytes:  plan.Quota.ValueString(),
		Groups:      groups,
		Enabled:     plan.Enabled.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating user", err.Error())
		return
	}
	pw := plan.Password
	r.fromAPI(ctx, out, &plan)
	plan.Password = pw
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteUser(ctx, state.UserID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting user", err.Error())
	}
}

func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("user_id"), req, resp)
}
