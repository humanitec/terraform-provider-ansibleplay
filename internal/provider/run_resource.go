// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
	ansiblePlaybookBinary string
}

type RunResourceModel struct {
	Hosts         types.List   `tfsdk:"hosts"`
	PlaybookFile  types.String `tfsdk:"playbook_file"`
	LastExecution types.String `tfsdk:"last_execution"`
}

func (r *RunResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_run"
}

func (r *RunResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Run resource",
		Attributes: map[string]schema.Attribute{
			"hosts": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "A list of hosts to run the playbook on.",
				Required:            true,
			},
			"playbook_file": schema.StringAttribute{
				MarkdownDescription: "A path to the playbook file to run.",
				Required:            true,
			},
			"last_execution": schema.StringAttribute{
				MarkdownDescription: "The last time the playbook was run.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *RunResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	if m, ok := req.ProviderData.(AnsiblePlayProviderModel); !ok {
		r.ansiblePlaybookBinary = m.AnsiblePlaybookBinary.ValueString()
	}
}

func (r *RunResource) execute(ctx context.Context, data RunResourceModel, checkOnly bool) error {
	hosts := make(map[string]interface{})
	for _, value := range data.Hosts.Elements() {
		hostAndJsonAttr := strings.SplitN(value.(basetypes.StringValue).ValueString(), " ", 2)
		attr := map[string]interface{}{}
		if len(hostAndJsonAttr) == 2 {
			if err := json.Unmarshal([]byte(hostAndJsonAttr[1]), &attr); err != nil {
				return fmt.Errorf("unable to parse host attributes for '%s': %w", hostAndJsonAttr[0], err)
			}
		}
		hosts[value.String()] = attr
	}
	raw, _ := yaml.Marshal(map[string]interface{}{
		"all": map[string]interface{}{
			"hosts": hosts,
		},
	})
	args := []string{
		data.PlaybookFile.ValueString(), "-i", "/dev/stdin", "-vv",
	}

	if checkOnly {
		args = append(args, "--check")
	}
	if r.ansiblePlaybookBinary == "" {
		r.ansiblePlaybookBinary = "ansible-playbook"
	}
	c := exec.CommandContext(ctx, r.ansiblePlaybookBinary, args...)
	c.Stdin = bytes.NewReader(raw)
	outBuffer := &bytes.Buffer{}
	errBuffer := &bytes.Buffer{}

	c.Stdout = outBuffer
	c.Stderr = errBuffer
	err := c.Run()

	tflog.Info(ctx, "ansible play output: "+outBuffer.String())

	if err != nil {
		return fmt.Errorf("ansible play failed: %w: %s", err, errBuffer.String())
	}

	return nil
}

func (r *RunResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RunResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if err := r.execute(ctx, data, false); err != nil {
		resp.Diagnostics.AddError("Error", err.Error())
	}
	data.LastExecution = types.StringValue(time.Now().Format(time.RFC3339))

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RunResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RunResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if err := r.execute(ctx, data, false); err != nil {
		resp.Diagnostics.AddError("Error", err.Error())
	}

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RunResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RunResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if err := r.execute(ctx, data, false); err != nil {
		resp.Diagnostics.AddError("Error", err.Error())
	}
	data.LastExecution = types.StringValue(time.Now().Format(time.RFC3339))

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
