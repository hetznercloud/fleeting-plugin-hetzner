# Quick start

After you [installed a GitLab runner](https://docs.gitlab.com/runner/install/), copy the configuration below to the GitLab runner configuration file and fill the missing information:

```toml
concurrent = 20 # max_instances * capacity_per_instance

check_interval = 0

[[runners]]
name = "hetzner-docker-autoscaler"
url = "https://gitlab.com" # TODO: Change me with the GitLab instance URL for the runner
token = "$RUNNER_TOKEN" # TODO: Change me with the runner authentication token

executor = "docker-autoscaler"

[runners.docker]
image = "busybox:latest"

[runners.autoscaler]
plugin = "hetznercloud/fleeting-plugin-hetzner:latest"

update_interval = "1m"
update_interval_when_expecting = "5s"

capacity_per_instance = 4
max_instances = 5
max_use_count = 0

# cloud-init>=23.4 returns an exit code 2 when the setup succeeded but some recoverable errors occurred.
# See https://cloudinit.readthedocs.io/en/latest/explanation/return_codes.html
instance_ready_command = "cloud-init status --wait || test $? -eq 2"

[runners.autoscaler.plugin_config]
name = "hetzner-docker-autoscaler"
token = "$HCLOUD_TOKEN" # TODO: Change me with the Hetzner Cloud authentication token

location = "hel1"
server_type = "cpx42"
image = "debian-12"
private_networks = []

user_data = """#cloud-config
package_update: true
package_upgrade: true

apt:
  sources:
    docker.list:
      source: deb [signed-by=$KEY_FILE] https://download.docker.com/linux/debian $RELEASE stable
      keyid: 9DC858229FC7DD38854AE2D88D81803C0EBFCD88

packages:
  - ca-certificates
  - docker-ce

swap:
  filename: /var/swap.bin
  size: auto
  maxsize: 4294967296 # 4GB
"""

[runners.autoscaler.connector_config]
# without private network, the instances are only reachable through their public addresses.
use_external_addr = true

[[runners.autoscaler.policy]]
periods = ["* * * * *"]
timezone = "Europe/Berlin"
idle_count = 0
idle_time = "50m"

[[runners.autoscaler.policy]]
periods = ["* 7-19 * * 1-5"]
timezone = "Europe/Berlin"
# idle_count refers to the number of jobs, not the number of instances.
idle_count = 8
idle_time = "50m"
```

Before starting the GitLab runner, you must [install the fleeting plugin](https://docs.gitlab.com/runner/fleet_scaling/fleeting.html#install-with-the-oci-registry-distribution), using the following command:

```sh
gitlab-runner fleeting install
```

Finally, you can start the GitLab runner.
