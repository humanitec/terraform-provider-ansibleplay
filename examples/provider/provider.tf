provider "ansibleplay" {
  # No configuration is required by default as long as the 'ansible-playbook' binary is available.
}

provider "ansibleplay" {
  alias = "with-specific-binary"

  # Some systems don't have ansible-playbook immediately accessible and it can be specified.
  ansible_playbook_binary = "/usr/bin/ansible-playbook"
}

provider "ansibleplay" {
  alias = "for-debugging"

  # If you're debugging SSH, host access, or playbook execution details. You can set the verbosity higher to see more
  # internal messages.
  verbosity = 3
}
