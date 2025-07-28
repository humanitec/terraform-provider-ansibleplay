// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"os"
	"slices"
	"testing"
	"time"

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
						tfjsonpath.New("last_execution"),
						knownvalue.StringFunc(func(v string) error {
							_, err := time.Parse(time.RFC3339, v)
							return err
						}),
					),
				},
			},
			// Update and Read testing
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
						tfjsonpath.New("last_execution"),
						knownvalue.StringFunc(func(v string) error {
							_, err := time.Parse(time.RFC3339, v)
							return err
						}),
					),
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccExampleResourceConfig(hosts []string, playbook string) string {
	f := hclwrite.NewEmptyFile()
	p := f.Body().AppendNewBlock("provider", []string{"ansibleplay"})
	p.Body().SetAttributeValue("ansible_playbook_binary", cty.StringVal("/Users/bmeier/lib/ansible/bin/ansible-playbook"))

	b := f.Body().AppendNewBlock("resource", []string{"ansibleplay_run", "test"})
	b.Body().SetAttributeValue("playbook_file", cty.StringVal(playbook))
	b.Body().SetAttributeValue("hosts", cty.TupleVal(slices.Collect[cty.Value](func(yield func(value cty.Value) bool) {
		for _, h := range hosts {
			yield(cty.StringVal(h))
		}
	})))
	buf := new(bytes.Buffer)
	_, _ = f.WriteTo(buf)
	return buf.String()
}
