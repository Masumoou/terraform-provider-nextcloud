package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Masumoou/terraform-provider-nextcloud/internal/client"
)

var _ resource.Resource = &OccCommandResource{}

func NewOccCommandResource() resource.Resource { return &OccCommandResource{} }

type OccCommandResource struct {
	client *client.Client
}

type occCommandModel struct {
	ID       types.String `tfsdk:"id"`
	Command  types.String `tfsdk:"command"`
	Args     types.List   `tfsdk:"args"`
	Triggers types.Map    `tfsdk:"triggers"`
	ExitCode types.Int64  `tfsdk:"exit_code"`
	Stdout   types.String `tfsdk:"stdout"`
	Stderr   types.String `tfsdk:"stderr"`
}

func (r *OccCommandResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_occ_command"
}

func (r *OccCommandResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Runs an arbitrary `occ` command through the nextcloud-agent, e.g. `maintenance:repair`, " +
			"`files:scan`, `db:add-missing-indices`, `maintenance:mimetype:update-db`, `maintenance:update:htaccess`. " +
			"Because occ commands are imperative rather than declarative, re-execution is controlled by the " +
			"`triggers` map: change any value there to force the command to run again on the next apply.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"command": schema.StringAttribute{
				Required:    true,
				Description: "occ command name, e.g. 'maintenance:repair'.",
			},
			"args": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			"triggers": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Arbitrary key/value pairs. Changing any value causes the command to re-run.",
			},
			"exit_code": schema.Int64Attribute{
				Computed: true,
			},
			"stdout": schema.StringAttribute{
				Computed: true,
			},
			"stderr": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *OccCommandResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *OccCommandResource) run(ctx context.Context, m *occCommandModel, diags *diagAppender) {
	var args []string
	m.Args.ElementsAs(ctx, &args, false)

	out, err := r.client.RunOcc(ctx, m.Command.ValueString(), args)
	if err != nil {
		diags.AddError("Error running occ command", err.Error())
		return
	}

	hasher := sha256.New()
	hasher.Write([]byte(m.Command.ValueString()))
	for _, a := range args {
		hasher.Write([]byte(a))
	}
	m.ID = types.StringValue(hex.EncodeToString(hasher.Sum(nil))[:16])
	m.ExitCode = types.Int64Value(int64(out.ExitCode))
	m.Stdout = types.StringValue(out.Stdout)
	m.Stderr = types.StringValue(out.Stderr)

	if out.ExitCode != 0 {
		diags.AddError("occ command exited non-zero", out.Stderr)
	}
}

func (r *OccCommandResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan occCommandModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.run(ctx, &plan, &diagAppender{&resp.Diagnostics})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read is intentionally a no-op beyond re-persisting state: occ commands are
// imperative actions, not queryable resources, so drift detection is driven
// entirely by the `triggers` map on Update.
func (r *OccCommandResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state occCommandModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OccCommandResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan occCommandModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.run(ctx, &plan, &diagAppender{&resp.Diagnostics})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OccCommandResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// No corresponding "undo" for an occ command; removing from state is enough.
}
