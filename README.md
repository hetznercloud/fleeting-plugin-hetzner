# Fleeting Plugin Hetzner

This is a [fleeting](https://gitlab.com/gitlab-org/fleeting/fleeting) plugin for [Hetzner Cloud](https://www.hetzner.com/cloud/).

> This plugin is experimental, breaking changes may occur without notice. You can expect
> the plugin to be experimental until we reach version 1.0.0.

[![Pipeline Status](https://gitlab.com/hetznercloud/fleeting-plugin-hetzner/badges/main/pipeline.svg)](https://gitlab.com/hetznercloud/fleeting-plugin-hetzner/commits/main)
[![Go Report Card](https://goreportcard.com/badge/gitlab.com/hetznercloud/fleeting-plugin-hetzner)](https://goreportcard.com/report/gitlab.com/hetznercloud/fleeting-plugin-hetzner)

## Building the plugin

To build the binary, ensure that your go version is up-to-date, and run the following:

```sh
make build
```

## Plugin Configuration

See the [configuration reference](docs/references/configuration.md) for the available configuration.

## Examples

### GitLab Runner

Below is a minimal GitLab runner configuration example that runs a `docker-autoscaler` executor using the Hetzner Cloud fleeting plugin:

```toml
concurrent = 20 # max_instances * capacity_per_instance

check_interval = 0

[[runners]]
name = "runner-docker-autoscaler0"
url = "https://gitlab.com"
token = "<your-gitlab-runner-authentication-token>"

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
name = "runner-docker-autoscaler0"
token = "<your-hetzner-cloud-token>"

location = "fsn1"
server_type = "cpx41"
image = "ubuntu-24.04"
private_networks = []

user_data = """#cloud-config
package_update: true
package_upgrade: true

apt:
  sources:
    docker.list:
      source: deb https://download.docker.com/linux/ubuntu $RELEASE stable
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
idle_time = "0s"

[[runners.autoscaler.policy]]
periods = ["* 7-19 * * 1-5"]
timezone = "Europe/Berlin"
# idle_count refers to the number of jobs, not the number of instances.
idle_count = 8
idle_time = "50m"
```

Before starting `gitlab-runner` with the configuration above, you must [install the fleeting plugin](https://docs.gitlab.com/runner/fleet_scaling/fleeting.html#install-with-the-oci-registry-distribution), using the following command:

```sh
gitlab-runner fleeting install
```

## Testing the plugin locally

To run the unit tests, run the following:

```sh
$ make test
```

For the integration tests to run, you need to export a Hetzner Cloud token in the `HCLOUD_TOKEN` environment variable before starting the tests.

### Testing the plugin with GitLab Runner

Sometimes, you want to test the whole plugin as its being executed by GitLab's Fleeting mechanism.
Use an approach like this:

1. Build the plugin by running the following:

   ```shell
   $ cd cmd/fleeting-plugin-hetzner
   $ go build
   ```

1. Set up the plugin in GitLab Runner's `config.toml` file using the approach described above, but
   update `plugin = "/path/to/fleeting-plugin-hetzner"` to point to your
   `cmd/fleeting-plugin-hetzner/fleeting-plugin-hetzner`

1. Run `gitlab-runner run` or similar, to run GitLab Runner interactively as a foreground process.

1. Make a CI job run using this runner, perhaps using special `tags:` or similar (to avoid breaking
   things for other CI jobs on the same GitLab installation).

## Creating a new release

**Follow [Semantic Versioning](https://semver.org/)**. Don't be afraid to bump the major version
when you are making changes to the public API.

We leverage the [releaser-pleaser](https://github.com/apricote/releaser-pleaser) tool to
prepare and cut releases. To cut a new release, you need to merge the Merge Request that
was prepared by releaser-pleaser.

### History

The project started out as a fork of the existing [gitlab-org/fleeting/plugins/aws](https://gitlab.com/hetznercloud/fleeting-plugin-hetzner/-/commit/5c71bcde58f5eb1272828bf34051b02510e7f0de) plugin, gradually replacing the AWS calls with calls to the [Hetzner Cloud API](https://github.com/hetznercloud/hcloud-go). To all the people involved in this initial work, **thanks a lot**!
