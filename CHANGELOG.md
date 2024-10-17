# Changelog

## [v0.5.0](https://gitlab.com/hetznercloud/fleeting-plugin-hetzner/-/tags/v0.5.0)

### Features

- clean instance if it failed to be created (!121)

### Bug Fixes

- deps: update gitlab.com/gitlab-org/fleeting/fleeting digest to a0ce7d6 (!118)

## [v0.4.0](https://gitlab.com/hetznercloud/fleeting-plugin-hetzner/-/tags/v0.4.0)

### Bug Fixes

- deps: update module github.com/hetznercloud/hcloud-go/v2 to v2.13.1 (!104)
- deps: update gitlab.com/gitlab-org/fleeting/fleeting digest to da5f142 (!97)
- client actions polling backoff (!99)
- use truncated exponential backoff with jitter when polling actions (!88)

## [v0.3.0](https://gitlab.com/hetznercloud/fleeting-plugin-hetzner/-/tags/v0.3.0)

### Features

- use available ip from pool (!81)

### Bug Fixes

- deps: update module github.com/hetznercloud/hcloud-go/v2 to v2.10.2 (!83)

## [v0.2.1](https://gitlab.com/hetznercloud/fleeting-plugin-hetzner/-/tags/v0.2.1)

### Bug Fixes

- improve log messages (!67)
- managed by label should be the software name (!64)

## [v0.2.0](https://gitlab.com/hetznercloud/fleeting-plugin-hetzner/-/tags/v0.2.0)

### Features

- use new hetzner cloud instance group abstraction (!53)
- rename access_token config to token (!58)
- remove disable_public_networks config field (!59)
- move project to gitlab.com/hetznercloud/fleeting-plugin-hetzner (!35)
- rename cloud_init_user_data settings to user_data (fleeting-plugin-hetzner/fleeting-plugin-hetzner!33)

### Bug Fixes

- deps: update gitlab.com/gitlab-org/fleeting/fleeting to 20240408 revision (fleeting-plugin-hetzner/fleeting-plugin-hetzner!29)
- wrong server location

## [v0.1.0](https://gitlab.com/hetznercloud/fleeting-plugin-hetzner/-/tags/v0.1.0)

### Features

- Support custom cloud-init user data (2c9a8877)
- Implement support for private networks (fc1fd091)
- Change SSH key generation slightly (f2c2a368)
- Delete servers on plugin shutdown (9217b4af)
- Implement basic functionality (429b6070)
- Start implementing some of the basic Hetzner API calls (a7144cd2)
- Remove WinRM support (2af26d66)

### Bug Fixes

- Fix arch info returned in ConnectInfo data (eb0f4334)
- Workaround for ServerStatusOff (d487cb43)
- Fix plugin configuration to work with config.toml data (5ea43110)
