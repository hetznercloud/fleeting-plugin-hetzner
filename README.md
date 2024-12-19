# [Fleeting Plugin Hetzner](https://gitlab.com/hetznercloud/fleeting-plugin-hetzner)

[![Pipeline Status](https://gitlab.com/hetznercloud/fleeting-plugin-hetzner/badges/main/pipeline.svg)](https://gitlab.com/hetznercloud/fleeting-plugin-hetzner/-/pipelines?scope=branches&ref=main)
[![Coverage](https://gitlab.com/hetznercloud/fleeting-plugin-hetzner/badges/main/coverage.svg?job=test)](https://gitlab.com/hetznercloud/fleeting-plugin-hetzner/-/pipelines?scope=branches&ref=main)
[![Go Report](https://goreportcard.com/badge/gitlab.com/hetznercloud/fleeting-plugin-hetzner)](https://goreportcard.com/report/gitlab.com/hetznercloud/fleeting-plugin-hetzner)
[![Releases](https://img.shields.io/gitlab/v/release/hetznercloud%2Ffleeting-plugin-hetzner)](https://gitlab.com/hetznercloud/fleeting-plugin-hetzner/-/releases)
![Maturity](https://img.shields.io/badge/maturity-general%20availability-yellow)
[![License](https://img.shields.io/gitlab/license/hetznercloud%2Ffleeting-plugin-hetzner)](https://gitlab.com/hetznercloud/fleeting-plugin-hetzner/-/blob/main/LICENSE)

A [fleeting](https://gitlab.com/gitlab-org/fleeting/fleeting) plugin for [Hetzner Cloud](https://www.hetzner.com/cloud/).

This tool was created to leverage GitLab's [Next Runner Auto-scaling Architecture](https://handbook.gitlab.com/handbook/engineering/architecture/design-documents/runner_scaling/) with [Hetzner Cloud](https://www.hetzner.com/cloud/), and take advantage of the new features that comes with it.

## Docs

- :rocket: See the [quick start guide](docs/guides/quickstart.md) to get you started.
- :book: See the [configuration reference](docs/references/configuration.md) for the available configuration.

For more information, see the [documentation](docs/).

## Releases

This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Please read the [changelog](CHANGELOG.md) before upgrading.

## Contribute

First and foremost, thank you! We appreciate that you want to contribute to this project.

When submitting your contribution to this project, make sure that:

- all the `pre-commit` hooks passes:

  ```bash
  pre-commit run --all
  ```

- all tests passes:

  ```bash
  make test
  ```

- any relevant documentation is updated with your changes.

## Development

### Building the plugin

To build the binary, ensure that your go version is up-to-date, and run the following:

```sh
$ make build
```

### Testing the plugin locally

To run the unit tests, run the following:

```sh
$ make test
```

For the integration tests to run, you need to export a Hetzner Cloud token in the `HCLOUD_TOKEN` environment variable before starting the tests.

#### Testing the plugin with GitLab Runner

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

### Setup a development environment

To setup a development environment, make sure you installed the following tools:

- [hcloud](https://github.com/hetznercloud/cli)
- [docker](https://www.docker.com/) (with the compose plugin)

1. Configure a `HCLOUD_TOKEN` and a `RUNNER_TOKEN` in your shell session.

> [!WARNING]
> The development environment creates Hetzner Cloud servers which will induce costs.

2. Run the development environment:

```sh
make -C dev up
```

> Also run this command to update the development environment with your latest code changes.

3. Check that the development environment is healthy:

```sh
docker compose logs
```

⚠️ Do not forget to clean up the development cluster once are finished:

```sh
make -C dev down
```

### Creating a new release

We leverage the [releaser-pleaser](https://github.com/apricote/releaser-pleaser) tool to
prepare and cut releases. To cut a new release, you need to merge the Merge Request that
was prepared by releaser-pleaser.

## History

The project started out as a fork of the existing [gitlab-org/fleeting/plugins/aws](https://gitlab.com/hetznercloud/fleeting-plugin-hetzner/-/commit/5c71bcde58f5eb1272828bf34051b02510e7f0de) plugin, gradually replacing the AWS calls with calls to the [Hetzner Cloud API](https://github.com/hetznercloud/hcloud-go). To all the people involved in this initial work, **thanks a lot**!
