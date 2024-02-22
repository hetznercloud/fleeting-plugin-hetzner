# Fleeting Plugin Hetzner

This is a [fleeting](https://gitlab.com/gitlab-org/fleeting/fleeting) plugin for [Hetzner
Cloud](https://www.hetzner.com/cloud/).

[![Pipeline Status](https://gitlab.com/hiboxsystems/fleeting-plugin-hetzner/badges/main/pipeline.svg)](https://gitlab.com/hiboxsystems/fleeting-plugin-hetzner/commits/main)
[![Go Report Card](https://goreportcard.com/badge/gitlab.com/hiboxsystems/fleeting-plugin-hetzner)](https://goreportcard.com/report/gitlab.com/hiboxsystems/fleeting-plugin-hetzner)

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

| Parameter      | Type   | Description                                                                                                                                             |
|----------------|--------|---------------------------------------------------------------------------------------------------------------------------------------------------------|
| `access_token` | string | The Hetzner Cloud API token to use. Generate this in the Hetzner Cloud Console, for the project in which you want the cloud CI instances to be created. |
| `location`     | string | The Hetzner location to use, from https://docs.hetzner.com/cloud/general/locations/                                                                     |
| `server_type`  | string | The server type to create, from https://docs.hetzner.com/cloud/servers/overview/                                                                        |
| `image`        | string | The operating system image to use. If you have the hcloud CLI installed, you can list available images using `hcloud image list --type system`.         |

### Connector config

The connector config is currently hardwired as follows:

| Parameter                | Value                                                                             |
|--------------------------|-----------------------------------------------------------------------------------|
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

  [runners.autoscaler.plugin_config] # plugin specific configuration (see plugin documentation)
    access_token      = "<insert-token-here>"
    location          = "hel1"
    server_type       = "cx11"
    image             = "ubuntu-22.04"

    # All instances created by this plugin will have their server name prefixed with this name
    name              = "my-docker-autoscaler-group"

  [[runners.autoscaler.policy]]
    idle_count = 1
    idle_time = "20m0s"
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
