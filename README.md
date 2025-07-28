# Terraform Provider for Ansible Execution

**NB:** This requires that the `ansible-playbook` executable exists on the system! This usually requires that Python/Pip has been used to install and set this up already. At the time of writing there is no Golang-native solution that we can easily compile into this provider.

```
terraform {
  required_providers {
    ansibleplay = {
      source  = "humanitec/ansibleplay"
      version = "~> 0.1"
    }
  }
}
```

Requirements:

- `ansible-playbook` executable
- Writable temporary directory for inventory files (currently Ansible doesn't support reading the inventory from a stdin source)

## Development

_This template repository is built on the [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework). The template repository built on the [Terraform Plugin SDK](https://github.com/hashicorp/terraform-plugin-sdk) can be found at [terraform-provider-scaffolding](https://github.com/hashicorp/terraform-provider-scaffolding). See [Which SDK Should I Use?](https://developer.hashicorp.com/terraform/plugin/framework-benefits) in the Terraform documentation for additional information._

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `make generate`. If you get a failure about "incompatible versions", use `GOOS=darwin GOARCH=amd64 go generate ./...` to set the platform specifically for example.

In order to run the full suite of Acceptance tests, run `make testacc`.

```shell
make testacc
```
