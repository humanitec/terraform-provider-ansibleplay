// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"encoding/json"
	"os"
	"slices"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
	"gopkg.in/yaml.v3"
)

func TestAccExampleResource(t *testing.T) {
	tf, err := os.CreateTemp(os.TempDir(), "terraform-provider-ansibleplay-test-*.yml")
	require.NoError(t, err)
	require.NoError(t, yaml.NewEncoder(tf).Encode([]interface{}{
		map[string]interface{}{
			"name":  "Example",
			"hosts": "all",
			"tasks": []interface{}{
				map[string]interface{}{
					"ansible.builtin.debug": map[string]interface{}{
						"msg": "Hello, World!",
					},
				},
			},
		},
	}))
	require.NoError(t, tf.Close())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccExampleResourceConfig([]string{`127.0.0.1 {"ansible_connection":"local"}`}, tf.Name()),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ansibleplay_run.test",
						tfjsonpath.New("hosts"),
						knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact(`127.0.0.1 {"ansible_connection":"local"}`)}),
					),
					statecheck.ExpectKnownValue(
						"ansibleplay_run.test",
						tfjsonpath.New("playbook_file"),
						knownvalue.StringExact(tf.Name()),
					),
					statecheck.ExpectKnownValue(
						"ansibleplay_run.test",
						tfjsonpath.New("extra_vars_json"),
						knownvalue.StringExact(`{"a":"b"}`),
					),
				},
			},
		},
	})
}

func testAccExampleResourceConfig(hosts []string, playbook string) string {
	f := hclwrite.NewEmptyFile()

	td := f.Body().AppendNewBlock("resource", []string{"terraform_data", "replacement"})
	td.Body().SetAttributeRaw("input", hclwrite.TokensForFunctionCall("filesha256", hclwrite.TokensForValue(cty.StringVal(playbook))))

	b := f.Body().AppendNewBlock("resource", []string{"ansibleplay_run", "test"})
	b.Body().SetAttributeValue("playbook_file", cty.StringVal(playbook))
	b.Body().SetAttributeValue("hosts", cty.TupleVal(slices.Collect[cty.Value](func(yield func(value cty.Value) bool) {
		for _, h := range hosts {
			yield(cty.StringVal(h))
		}
	})))
	raw, _ := json.Marshal(map[string]interface{}{"a": "b"})
	b.Body().SetAttributeValue("extra_vars_json", cty.StringVal(string(raw)))
	lff := b.Body().AppendNewBlock("lifecycle", nil)
	lff.Body().SetAttributeRaw("replace_triggered_by", hclwrite.TokensForTuple([]hclwrite.Tokens{
		hclwrite.TokensForTraversal([]hcl.Traverser{hcl.TraverseRoot{Name: "terraform_data"}, hcl.TraverseAttr{Name: "replacement"}}),
	}))
	buf := new(bytes.Buffer)
	_, _ = f.WriteTo(buf)
	return buf.String()
}
