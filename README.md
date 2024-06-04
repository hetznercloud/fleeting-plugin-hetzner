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

See the [configuration reference](docs/configuration.md) for the available configuration.

## Examples

### GitLab Runner

Below is a minimal GitLab runner configuration example that runs a `docker-autoscaler` executor using the Hetzner Cloud fleeting plugin:

```toml
concurrent = 20 # max_instances * capacity_per_instance

check_interval = 0

[[runners]]
name = "runner-docker-autoscaler0"
url = "https://gitlab.com"
id = <your-gitlab-project-id>
token = "<your-gitlab-runner-authentication-token>"

executor = "docker-autoscaler"

[runners.docker]
image = "busybox:latest"

[runners.autoscaler]
plugin = "hetznercloud/fleeting-plugin-hetzner:latest"

capacity_per_instance = 4
max_instances = 5
max_use_count = 0

// cloud-init>=23.4 returns an exit code 2 when the setup succeeded but some recoverable errors occurred.
// See https://cloudinit.readthedocs.io/en/latest/explanation/return_codes.html
instance_ready_command = "cloud-init status --wait || test $? -eq 2"

[runners.autoscaler.plugin_config]
name = "runner-docker-autoscaler0"
token = "<your-hetzner-cloud-token>"

location = "fsn1"
server_type = "cpx41"
image = "ubuntu-24.04"
private_networks = []

user_data_file = """#cloud-config
package_update: true
package_upgrade: true

apt:
  sources:
    docker.list:
      source: deb https://download.docker.com/linux/ubuntu $RELEASE stable
      keyid: 9DC858229FC7DD38854AE2D88D81803C0EBFCD88

    gitlab-runner.list:
      source: deb https://packages.gitlab.com/runner/gitlab-runner/ubuntu/ $RELEASE main
      keyid: F6403F6544A38863DAA0B6E03F01618A51312F3F

packages:
  - ca-certificates
  - docker-ce
  - git
  - gitlab-runner

swap:
  filename: /var/swap.bin
  size: auto
  maxsize: 4294967296 # 4GB
"""

[runners.autoscaler.connector_config]
use_external_addr = true

[[runners.autoscaler.policy]]
periods = ["* 7-19 * * mon-fri"]
idle_count = 1
idle_time = "20m0s"

[[runners.autoscaler.policy]]
periods = ["* * * * *"]
idle_count = 0
idle_time = "0s"
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

1. Make sure the `VERSION` file is up-to-date. This file is typically edited at the beginning of the
   new release cycle (in other words "work towrads version x.y.z" rather than "release version x.y.z")
2. If not, commit the changes to that file, using a commit message in line with this: `git commit VERSION -m "Bump version to v0.2.0"`. Make sure to `git push` the commit as well.
3. Run `make do-release`. This creates a tag, which in turns triggers some CI logic in
   [`.gitlab/ci/release.gitlab-ci.yml`](.gitlab/ci/release.gitlab-ci.yml) which creates a GitLab
   release.
4. Edit the release notes in
   https://gitlab.com/fleeting-plugin-hetzner/fleeting-plugin-hetzner/-/releases. For inspiration, use
   https://gitlab.com/fleeting-plugin-hetzner/fleeting-plugin-hetzner/-/releases/v0.1.0 as an example.
   The "Full changelog" content can be retrieved using `make release-notes PREVIOUS_VERSION=0.1.0`
   (replace the version number with the "real" version number of the previous release)

### History

The project started out as a fork of the existing [gitlab-org/fleeting/plugins/aws](https://gitlab.com/hetznercloud/fleeting-plugin-hetzner/-/commit/5c71bcde58f5eb1272828bf34051b02510e7f0de) plugin, gradually replacing the AWS calls with calls to the [Hetzner Cloud API](https://github.com/hetznercloud/hcloud-go). To all the people involved in this initial work, **thanks a lot**!
