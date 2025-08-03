resource "ansibleplay_run" "simple" {
  # Hosts are set as either hostnames or ip addresses.
  hosts = ["alice.hosts.com", "bob.hosts.com", "10.0.0.1"]

  # The playbook must also be set to a relative path.
  playbook = "./example.yml"
}

resource "ansibleplay_run" "host-vars" {
  # Hosts can contain host variables encoded as json after a space.
  hosts = ["10.0.0.1 {\"ansible_user\": \"nobody\"}"]

  playbook = "./example.yml"
}

resource "ansibleplay_run" "extra-vars" {
  hosts    = ["alice.hosts.com", "bob.hosts.com"]
  playbook = "./example.yml"

  # Extra vars to the playbook can be set as a json encoded deeply nested structure.
  extra_vars_json = jsonencode({
    app_name     = "example"
    service_port = 8080
  })
}
