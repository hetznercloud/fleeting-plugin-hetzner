# Fleeting Plugin Hetzner

This is a [fleeting](https://gitlab.com/gitlab-org/fleeting/fleeting) plugin for [Hetzner
Cloud](https://www.hetzner.com/cloud/).

> This plugin is experimental, breaking changes may occur without notice. You can expect
> the plugin to be experimental until we reach version 1.0.0.

[![Pipeline Status](https://gitlab.com/hetznercloud/fleeting-plugin-hetzner/badges/main/pipeline.svg)](https://gitlab.com/hetznercloud/fleeting-plugin-hetzner/commits/main)
[![Go Report Card](https://goreportcard.com/badge/gitlab.com/hetznercloud/fleeting-plugin-hetzner)](https://goreportcard.com/report/gitlab.com/hetznercloud/fleeting-plugin-hetzner)

The project started out as a fork of the existing
[fleeting-plugin-aws](https://gitlab.com/gitlab-org/fleeting/fleeting-plugin-aws) plugin, gradually
replacing the AWS calls with calls to the [Hetzner Cloud
API](https://github.com/hetznercloud/hcloud-go).

## Building the plugin

To generate the binary, ensure `$GOPATH/bin` is on your PATH, then use `go build`:

```shell
cd cmd/fleeting-plugin-hetzner/
go build
```

If you are managing go versions with asdf, run this after generating the binary:

```shell
asdf reshim
```

## Plugin Configuration

The following parameters are supported:

| Parameter      | Type   | Description                                                                                                                                                                   |
| -------------- | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `access_token` | string | The Hetzner Cloud API token to use. Generate this in the Hetzner Cloud Console, for the project in which you want the cloud CI instances to be created.                       |
| `location`     | string | The Hetzner location to use, from https://docs.hetzner.com/cloud/general/locations/                                                                                           |
| `server_type`  | string | The server type to create, from https://docs.hetzner.com/cloud/servers/overview/                                                                                              |
| `image`        | string | The operating system image to use. If you have the hcloud CLI installed, you can list available images using `hcloud image list --type system`.                               |
| `name`         | string | All instances created by this plugin will have their server name prefixed with this name. They will also have a special label `instance-group` with the value specified here. |

### Connector config

The connector config is currently hardwired as follows:

| Parameter                | Value                                                                             |
| ------------------------ | --------------------------------------------------------------------------------- |
| `os`                     | Only `linux` is supported.                                                        |
| `protocol`               | `ssh` (`winrm` is not supported)                                                  |
| `username`               | `root`; the Hetzner Cloud API does not seem to allow overriding this.             |
| `use_static_credentials` | `false`; a unique SSH private/public key will be created for each server created. |
| `key_path`               | None.                                                                             |

## Examples

### GitLab Runner

GitLab Runner has examples on using the other plugins for the [Instance
executor](https://docs.gitlab.com/runner/executors/instance.html#examples) and [Docker Autoscaler
executor](https://docs.gitlab.com/runner/executors/docker_autoscaler.html#examples). Here is an
incomplete example of how to use this plugin with the `docker-autoscaler` executor, starting from
the `runners.docker` node. Both `runners.docker` and `runners.autoscaler` need to exist, because the
autoscaler will otherwise complaining about `"missing docker configuration"`.

```toml
# ...
  [runners.docker]
    tls_verify = false
    image = "busybox"

  [runners.autoscaler]
    plugin = "/path/to/fleeting-plugin-hetzner"

    capacity_per_instance = 1
    max_use_count = 1
    max_instances = 10

    # Set this if you want to connect using a public address. If you use private_networks, you can leave
    # this out (the default is 'false', i.e. connect using internal address only)
    [runners.autoscaler.connector_config]
      use_external_addr = true

    [runners.autoscaler.plugin_config] # plugin specific configuration (see plugin documentation)
      access_token      = "<insert-token-here>"
      location          = "hel1"
      server_type       = "cx11"

      # docker-ce is an "app" image provided by Hetzner which is based on Ubuntu 22.04, but provides
      # Docker CE preinstalled: https://docs.hetzner.com/cloud/apps/list/docker-ce/
      #
      # You could also use another image here, but it must have Docker installed.
      image             = "docker-ce"

      # All instances created by this plugin will have their server name prefixed with this name
      name              = "my-docker-autoscaler-group"

      # Public IPv4 and IPv6 are enabled on Hetzner by default. These can be disabled below, but you must
      # add one or more private networks in that case; otherwise the Hetzner cloud API will return errors
      # when we try to create instances. It is also possible to use public and private networks
      # simultaneously.
      disable_public_networks   = ["ipv4", "ipv6"]
      private_networks          = ["hetzner-cloud-ci-network"]

      # If you like to, you can specify a cloud-init configuration using one of the following forms (note,
      # both cannot be used simultaneously). The first example below is taken from
      # https://cloudinit.readthedocs.io/en/latest/reference/examples.html. Remember that the cloud-init
      # user-data must begin with #cloud-config, otherwise the file will be silently ignored.
      user_data = """
#cloud-config
users:
- name: ansible
  gecos: Ansible User
  groups: users,admin,wheel
  sudo: ALL=(ALL) NOPASSWD:ALL
  shell: /bin/bash
  lock_passwd: true
  ssh_authorized_keys:
    - "ssh-rsa AAAAB3NzaC1..."
    """
      user_data_file = "/path/to/cloud-init/user-data.yml"

    [[runners.autoscaler.policy]]
      idle_count        = 1
      idle_time         = "20m0s"
```

## Testing the plugin locally

Running the unit tests is easy, and this is also done for each MR/commit to `main` on GitLab:

```shell
$ go test
```

Extending this to also run the integration test requires that you provide a valid
`FLEETING_PLUGIN_HETZNER_TOKEN` environment variable. Something like this (or put it in your
`~/.bashrc` or similar):

```shell
$ FLEETING_PLUGIN_HETZNER_TOKEN=foo go test
```

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

### Project history

This project is based on [gitlab-org/fleeting/fleeting-plugin-aws](https://gitlab.com/fleeting-plugin-hetzner/fleeting-plugin-hetzner/-/commit/5c71bcde58f5eb1272828bf34051b02510e7f0de). To all the people involved in this initial work, _thanks a lot_!
