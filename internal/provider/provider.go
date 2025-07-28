// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &AnsiblePlayProvider{}
var _ provider.ProviderWithFunctions = &AnsiblePlayProvider{}
var _ provider.ProviderWithEphemeralResources = &AnsiblePlayProvider{}

type AnsiblePlayProvider struct {
	version string
}

// AnsiblePlayProviderModel describes the provider data model.
type AnsiblePlayProviderModel struct {
	AnsiblePlaybookBinary types.String `tfsdk:"ansible_playbook_binary"`
	Verbosity             types.Int32  `tfsdk:"verbosity"`
}

func (p *AnsiblePlayProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ansibleplay"
	resp.Version = p.version
}

func (p *AnsiblePlayProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"ansible_playbook_binary": schema.StringAttribute{
				MarkdownDescription: "The path to the Ansible playbook binary. This will be discovered automatically if not provided.",
				Optional:            true,
			},
			"verbosity": schema.Int32Attribute{
				MarkdownDescription: "The verbosity level to use when running the playbook.",
				Optional:            true,
			},
		},
	}
}

func (p *AnsiblePlayProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data AnsiblePlayProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.AnsiblePlaybookBinary.IsNull() {
		_, err := exec.LookPath("ansible-playbook")
		if err != nil {
			resp.Diagnostics.AddError("ansible-playbook binary not found in PATH", err.Error())
			return
		}
	} else if _, err := os.Stat(data.AnsiblePlaybookBinary.ValueString()); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("ansible-playbook binary '%s' could not be stat'd", data.AnsiblePlaybookBinary.ValueString()), err.Error())
		return
	}

	resp.DataSourceData = data
	resp.ResourceData = data
}

func (p *AnsiblePlayProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewRunResource,
	}
}

func (p *AnsiblePlayProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

func (p *AnsiblePlayProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *AnsiblePlayProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &AnsiblePlayProvider{
			version: version,
		}
	}
}
