// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"os/exec"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"gopkg.in/yaml.v3"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &RunResource{}

func NewRunResource() resource.Resource {
	return &RunResource{}
}

type RunResource struct {
	providerModel AnsiblePlayProviderModel
}

type RunResourceModel struct {
	Id           types.Int64  `tfsdk:"id"`
	Hosts        types.List   `tfsdk:"hosts"`
	PlaybookFile types.String `tfsdk:"playbook_file"`
	ExtraVars    types.String `tfsdk:"extra_vars_json"`
}

func (r *RunResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_run"
}

func (r *RunResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The run resource allows you to run an ansible playbook. The run will attempt to execute the given
playbook_file on the set of hosts with any extra_vars provided as json.

Note, this resource will not automatically re-run if the playbook file has changed. And may not run if there have been
no changes to the hosts or vars either. To ensure the run is always executed, use the ` + "`" + `lifecycle.replace_triggered_by` + "`" + `
attribute to re-execute the run based on the hash of the playbook file or timestamp.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "This is set to a random value at create time.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"hosts": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "A list of hosts to run the playbook on. Each host (an ip or hostname) may be followed by a space and a JSON object of host attributes.",
				Required:            true,
			},
			"playbook_file": schema.StringAttribute{
				MarkdownDescription: "A path to the playbook file to run.",
				Required:            true,
			},
			"extra_vars_json": schema.StringAttribute{
				MarkdownDescription: "A json-encoded map of extra variables to pass to the playbook.",
				Optional:            true,
			},
		},
	}
}

func (r *RunResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if r.providerModel, ok = req.ProviderData.(AnsiblePlayProviderModel); !ok {
		resp.Diagnostics.AddError("failed to convert provider data to AnsiblePlayProviderModel", "provider data is not AnsiblePlayProviderModel")
		return
	}
}

func (r *RunResource) execute(ctx context.Context, data RunResourceModel) error {
	hosts := make(map[string]interface{})
	for _, value := range data.Hosts.Elements() {
		hv, _ := value.(basetypes.StringValue)
		hostAndJsonAttr := strings.SplitN(hv.ValueString(), " ", 2)
		attr := map[string]interface{}{}
		if len(hostAndJsonAttr) == 2 {
			if err := json.Unmarshal([]byte(hostAndJsonAttr[1]), &attr); err != nil {
				return fmt.Errorf("unable to parse host attributes for '%s': %w", hostAndJsonAttr[0], err)
			}
		}
		hosts[hostAndJsonAttr[0]] = attr
	}

	tf, err := os.CreateTemp(os.TempDir(), "inventory-*.yml")
	if err != nil {
		return fmt.Errorf("failed to create temporary inventory file: %w", err)
	}
	if err := yaml.NewEncoder(tf).Encode(map[string]interface{}{
		"all": map[string]interface{}{
			"hosts": hosts,
		},
	}); err != nil {
		return fmt.Errorf("failed to write temporary inventory file: %w", err)
	}
	if err := tf.Close(); err != nil {
		return fmt.Errorf("failed to close temporary inventory file: %w", err)
	}
	args := []string{
		data.PlaybookFile.ValueString(), "-i", tf.Name(),
	}

	if !data.ExtraVars.IsNull() {
		args = append(args, "--extra-vars", data.ExtraVars.ValueString())
	}

	if v := r.providerModel.Verbosity.ValueInt32(); v > 0 {
		args = append(args, "-"+strings.Repeat("v", int(v)))
	}
	binary := r.providerModel.AnsiblePlaybookBinary.ValueString()
	if binary == "" {
		binary = "ansible-playbook"
	}
	c := exec.CommandContext(ctx, binary, args...)
	outBuffer := &bytes.Buffer{}
	errBuffer := &bytes.Buffer{}

	c.Stdout = outBuffer
	c.Stderr = errBuffer
	err = c.Run()

	tflog.Info(ctx, "ansible play output: "+outBuffer.String())

	if err != nil {
		return fmt.Errorf("ansible play failed: %w: %s", err, errBuffer.String())
	}

	return nil
}

func (r *RunResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RunResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if err := r.execute(ctx, data); err != nil {
		resp.Diagnostics.AddError("Error", err.Error())
	}
	if resp.Diagnostics.HasError() {
		return
	}
	data.Id = types.Int64Value(int64(rand.Int()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RunResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RunResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.ExtraVars.IsNull() && !data.ExtraVars.IsUnknown() {
		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(data.ExtraVars.ValueString()), &raw); err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("extra_vars"), "extra_vars is not valid", "expected a valid json object")
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RunResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RunResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if err := r.execute(ctx, data); err != nil {
		resp.Diagnostics.AddError("Error", err.Error())
	}

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RunResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RunResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
