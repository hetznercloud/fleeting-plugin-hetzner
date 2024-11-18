# Setup a GitLab CI test infrastructure from scratch

This tutorial serves a learning material to setup a GitLab CI test infrastructure from scratch, using the Hetzner Cloud Fleeting plugin.

[TOC]

## Requirements

Before we start, make sure that you:

- know how to use a command line interface,
- have a [Hetzner Cloud account](https://accounts.hetzner.com/login),
- have the [`hcloud` CLI](https://github.com/hetznercloud/cli) installed on your device.

## 1. Create the infrastructure

### 1.1. Setup a Hetzner Cloud project

Let's start by creating a new Hetzner Cloud project named `gitlab-ci` using the Hetzner Cloud Console: https://console.hetzner.cloud/projects

Using a dedicated Hetzner Cloud project for your CI workloads only is recommended as it will reduce the risk of running into project rate limits, and possibly break your other workloads.

Next, for the fleeting plugin to communicate with Hetzner Cloud API, we [generate a new API token](https://docs.hetzner.com/cloud/api/getting-started/generating-api-token/) with read and write access.

Save the API token in a new `hcloud` CLI context, named after the project:

```sh
hcloud context create gitlab-ci
```

<details><summary>Output</summary>

```
Token:
Context gitlab-ci created and activated
```

</details>

### 1.2. Upload our public ssh key

Then, we upload our public ssh key to be able to connect to the future instances without
relying on a password based authentication.

```sh
hcloud ssh-key create --name dev --public-key-from-file ~/.ssh/id_ed25519.pub
```

<details><summary>Output</summary>

```
SSH key 22155019 created
```

</details>

### 1.3. Create the `runner-manager` instance

The GitLab Runner Manager will be responsible for scaling up and down the instances, executing your CI jobs on the instances, and forwarding the jobs logs to your GitLab instance.

We create a single `runner-manager` server that will be used as our GitLab Runner Manager:

```sh
hcloud server create --name runner-manager --image debian-12 --type cpx11 --location hel1 --ssh-key dev --label runner=
```

<details><summary>Output</summary>

```
 ✓ Waiting for create_server       100% 8.6s (server: 55574479)
 ✓ Waiting for start_server        100% 8.6s (server: 55574479)
Server 55574479 created
IPv4: 65.109.174.102
IPv6: 2a01:4f9:c012:1a18::1
IPv6 Network: 2a01:4f9:c012:1a18::/64
```

</details>

### 1.4. Configure firewalls

To increase the security of our CI instances, we create a firewall that allows only ICMP and SSH access to the instances:

```sh
hcloud firewall create --name runner --rules-file <(echo '[{
  "description": "allow icmp from everywhere",
  "direction": "in",
  "source_ips": ["0.0.0.0/0", "::/0"],
  "protocol": "icmp"
},
{
  "description": "allow ssh from everywhere",
  "direction": "in",
  "source_ips": ["0.0.0.0/0", "::/0"],
  "protocol": "tcp",
  "port": "22"
}]')
```

<details><summary>Output</summary>

```
 ✓ Waiting for set_firewall_rules  100% 0s (firewall: 1733905)
Firewall 1733905 created
```

</details>

After creating the firewall, we will apply the firewall on the servers that match a label selector, in our case `runner`:

```sh
hcloud firewall apply-to-resource runner --type label_selector --label-selector runner
```

<details><summary>Output</summary>

```
 ✓ Waiting for apply_firewall      100% 0s (firewall: 1733905)
Firewall 1733905 applied to resource
```

</details>

### 1.5. Overview

We just finished to create the base of the infrastructure, you may have an overview using the `hcloud` CLI:

```sh
hcloud all list
```

<details><summary>Output</summary>

```
SERVERS
---
ID         NAME              STATUS    IPV4              IPV6                      PRIVATE NET   DATACENTER   AGE
55574479   runner-manager    running   65.109.174.102    2a01:4f9:c012:1a18::/64   -             hel1-dc2     6m

PRIMARY IPS
---
ID         TYPE   NAME                  IP                        ASSIGNEE                 DNS                                             AUTO DELETE   AGE
74302282   ipv4   primary_ip-74302282   65.109.174.102            Server runner-manager    static.102.174.109.65.clients.your-server.de    yes           6m
74302283   ipv6   primary_ip-74302283   2a01:4f9:c012:1a18::/64   Server runner-manager    -                                               yes           6m

FIREWALLS
---
ID        NAME     RULES COUNT   APPLIED TO COUNT
1733905   runner   2 Rules       0 Servers | 1 Label Selector

SSH KEYS
---
ID         NAME    FINGERPRINT                                       AGE
22499499   dev     2b:9f:a0:6d:01:12:a4:4d:2b:27:02:34:56:bf:fe:5f   10m
```

</details>

Now that the base infrastructure has been created, we will deploy the `gitlab-runner` software that will schedule our CI jobs.

## 2. Deploy the GitLab Runner Manager

Every step in this section must be executed on the server that was created in the step [1.3](#13-create-the-runner-manager-instance). To connect to the `runner-manager`, run the following command:

```sh
hcloud server ssh runner-manager
```

<details><summary>Output</summary>

```
Linux runner-manager 6.1.0-26-amd64 #1 SMP PREEMPT_DYNAMIC Debian 6.1.112-1 (2024-09-30) x86_64

The programs included with the Debian GNU/Linux system are free software;
the exact distribution terms for each program are described in the
individual files in /usr/share/doc/*/copyright.

Debian GNU/Linux comes with ABSOLUTELY NO WARRANTY, to the extent
permitted by applicable law.
root@runner-manager:~#
```

</details>

### 2.1. Install `gitlab-runner`

To install the `gitlab-runner` package, we must add the GitLab Runner apt package repository:

```sh
curl -sSL "https://packages.gitlab.com/install/repositories/runner/gitlab-runner/script.deb.sh" | sudo bash
```

<details><summary>Output</summary>

```
Detected operating system as debian/bookworm.
Checking for curl...
Detected curl...
Checking for gpg...
Detected gpg...
Running apt-get update... done.
Installing debian-archive-keyring which is needed for installing
apt-transport-https on many Debian systems.
Installing apt-transport-https... done.
Installing /etc/apt/sources.list.d/runner_gitlab-runner.list...done.
Importing packagecloud gpg key... done.
Running apt-get update... done.

The repository is setup! You can now install packages.
```

</details>

```sh
sudo apt install gitlab-runner
```

<details><summary>Output</summary>

```
Reading package lists... Done
Building dependency tree... Done
Reading state information... Done
The following additional packages will be installed:
  git git-man liberror-perl libgdbm-compat4 libperl5.36 perl perl-modules-5.36
Suggested packages:
  git-daemon-run | git-daemon-sysvinit git-doc git-email git-gui gitk gitweb git-cvs git-mediawiki git-svn docker-engine perl-doc libterm-readline-gnu-perl | libterm-readline-perl-perl make libtap-harness-archive-perl
The following NEW packages will be installed:
  git git-man gitlab-runner liberror-perl libgdbm-compat4 libperl5.36 perl perl-modules-5.36
0 upgraded, 8 newly installed, 0 to remove and 48 not upgraded.
Need to get 550 MB of archives.
After this operation, 708 MB of additional disk space will be used.
Do you want to continue? [Y/n] y
Get:1 http://deb.debian.org/debian bookworm/main amd64 perl-modules-5.36 all 5.36.0-7+deb12u1 [2,815 kB]
Get:2 http://deb.debian.org/debian bookworm/main amd64 libgdbm-compat4 amd64 1.23-3 [48.2 kB]
Get:3 http://deb.debian.org/debian bookworm/main amd64 libperl5.36 amd64 5.36.0-7+deb12u1 [4,218 kB]
Get:4 http://deb.debian.org/debian bookworm/main amd64 perl amd64 5.36.0-7+deb12u1 [239 kB]
Get:5 http://deb.debian.org/debian bookworm/main amd64 liberror-perl all 0.17029-2 [29.0 kB]
Get:6 http://deb.debian.org/debian bookworm/main amd64 git-man all 1:2.39.5-0+deb12u1 [2,054 kB]
Get:7 http://deb.debian.org/debian bookworm/main amd64 git amd64 1:2.39.5-0+deb12u1 [7,256 kB]
Get:8 https://packages.gitlab.com/runner/gitlab-runner/debian bookworm/main amd64 gitlab-runner amd64 17.5.3-1 [533 MB]
Fetched 550 MB in 4s (127 MB/s)
Selecting previously unselected package perl-modules-5.36.
(Reading database ... 34132 files and directories currently installed.)
Preparing to unpack .../0-perl-modules-5.36_5.36.0-7+deb12u1_all.deb ...
Unpacking perl-modules-5.36 (5.36.0-7+deb12u1) ...
Selecting previously unselected package libgdbm-compat4:amd64.
Preparing to unpack .../1-libgdbm-compat4_1.23-3_amd64.deb ...
Unpacking libgdbm-compat4:amd64 (1.23-3) ...
Selecting previously unselected package libperl5.36:amd64.
Preparing to unpack .../2-libperl5.36_5.36.0-7+deb12u1_amd64.deb ...
Unpacking libperl5.36:amd64 (5.36.0-7+deb12u1) ...
Selecting previously unselected package perl.
Preparing to unpack .../3-perl_5.36.0-7+deb12u1_amd64.deb ...
Unpacking perl (5.36.0-7+deb12u1) ...
Selecting previously unselected package liberror-perl.
Preparing to unpack .../4-liberror-perl_0.17029-2_all.deb ...
Unpacking liberror-perl (0.17029-2) ...
Selecting previously unselected package git-man.
Preparing to unpack .../5-git-man_1%3a2.39.5-0+deb12u1_all.deb ...
Unpacking git-man (1:2.39.5-0+deb12u1) ...
Selecting previously unselected package git.
Preparing to unpack .../6-git_1%3a2.39.5-0+deb12u1_amd64.deb ...
Unpacking git (1:2.39.5-0+deb12u1) ...
Selecting previously unselected package gitlab-runner.
Preparing to unpack .../7-gitlab-runner_17.5.3-1_amd64.deb ...
Unpacking gitlab-runner (17.5.3-1) ...
Setting up perl-modules-5.36 (5.36.0-7+deb12u1) ...
Setting up libgdbm-compat4:amd64 (1.23-3) ...
Setting up git-man (1:2.39.5-0+deb12u1) ...
Setting up libperl5.36:amd64 (5.36.0-7+deb12u1) ...
Setting up perl (5.36.0-7+deb12u1) ...
Setting up liberror-perl (0.17029-2) ...
Setting up git (1:2.39.5-0+deb12u1) ...
Setting up gitlab-runner (17.5.3-1) ...
GitLab Runner: creating gitlab-runner...
Home directory skeleton not used
Runtime platform                                    arch=amd64 os=linux pid=2230 revision=12030cf4 version=17.5.3
gitlab-runner: the service is not installed
Runtime platform                                    arch=amd64 os=linux pid=2237 revision=12030cf4 version=17.5.3
gitlab-ci-multi-runner: the service is not installed
Runtime platform                                    arch=amd64 os=linux pid=2256 revision=12030cf4 version=17.5.3
Runtime platform                                    arch=amd64 os=linux pid=2301 revision=12030cf4 version=17.5.3
INFO: Docker installation not found, skipping clear-docker-cache
Processing triggers for man-db (2.11.2-2) ...
Processing triggers for libc-bin (2.36-9+deb12u8) ...
```

</details>

You can find more details on the [GitLab Runner installation documentation](https://docs.gitlab.com/runner/install/linux-repository.html#installing-gitlab-runner) from which the above commands were copied.

### 2.2. Get a runner authentication token

For the GitLab Runner to retrieve jobs from your GitLab instance, we must get a runner authentication token. You may choose between [an instance runner](https://docs.gitlab.com/ee/ci/runners/runners_scope.html#create-an-instance-runner-with-a-runner-authentication-token), [a group runner](https://docs.gitlab.com/ee/ci/runners/runners_scope.html#create-a-group-runner-with-a-runner-authentication-token) or [a project runner](https://docs.gitlab.com/ee/ci/runners/runners_scope.html#create-a-project-runner-with-a-runner-authentication-token).

### 2.3. Configure the Fleeting plugin

Open the `/etc/gitlab-runner/config.toml` file, and replace the content with the configuration below:

```toml
concurrent = 10

log_level = "info"
log_format = "text"

[[runners]]
name = "hetzner-docker-autoscaler"
url = "https://gitlab.com" # TODO: Change me with the GitLab instance url for the runner
token = "$RUNNER_TOKEN" # TODO: Change me with the runner authentication token

executor = "docker-autoscaler"

[runners.docker]
image = "alpine:latest"

[runners.autoscaler]
plugin = "hetznercloud/fleeting-plugin-hetzner:latest"

capacity_per_instance = 4
max_instances = 5
max_use_count = 0

instance_ready_command = "cloud-init status --wait || test $? -eq 2"

[runners.autoscaler.plugin_config]
name = "runner-docker-autoscaler"
token = "$HCLOUD_TOKEN" # TODO: Change me with the Hetzner Cloud authentication token

location = "hel1"
server_type = "cpx21"
image = "debian-12"

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
use_external_addr = true

[[runners.autoscaler.policy]]
periods = ["* * * * *"]
timezone = "Europe/Berlin" # TODO: Change me with your timezone
idle_count = 8
idle_time = "1h"
```

Make sure that you updated the values for the runner url, runner token, and the Hetzner Cloud token.

```sh
gitlab-runner fleeting install
```

<details><summary>Output</summary>

```
Runtime platform                                    arch=amd64 os=linux pid=2524 revision=12030cf4 version=17.5.3
runner: 11Qjxy-Gi, plugin: hetznercloud/fleeting-plugin-hetzner:latest, path: /root/.config/fleeting/plugins/registry.gitlab.com/hetznercloud/fleeting-plugin-hetzner/0.6.0/plugin
```

</details>

```sh
systemctl restart gitlab-runner
```

```sh
systemctl status --output=cat --no-pager gitlab-runner
```

<details><summary>Output</summary>

```
● gitlab-runner.service - GitLab Runner
     Loaded: loaded (/etc/systemd/system/gitlab-runner.service; enabled; preset: enabled)
     Active: active (running) since Wed 2024-11-13 09:57:24 UTC; 6min ago
   Main PID: 2587 (gitlab-runner)
      Tasks: 16 (limit: 2251)
     Memory: 35.6M
        CPU: 1.880s
     CGroup: /system.slice/gitlab-runner.service
             ├─2587 /usr/bin/gitlab-runner run --config /etc/gitlab-runner/config.toml --working-directory /home/gitlab-runner --service gitlab-runner --user gitlab-runner
             └─2595 /root/.config/fleeting/plugins/registry.gitlab.com/hetznercloud/fleeting-plugin-hetzner/0.6.0/plugin

time="2024-11-13T09:57:25Z" level=info msg="plugin initialized" build info="sha=85c314ff; ref=refs/pipelines/1528252336; go=go1.23.2; built_at=2024-11-05T15:20:21+0000; os_arch=linux/amd64" runner=11Qjxy-Gi subsystem=taskscaler version=v0.6.0
time="2024-11-13T09:57:26Z" level=info msg="required scaling change" capacity-info="instance_count:0,max_instance_count:5,acquired:0,unavailable_capacity:0,pending:0,reserved:0,idle_count:8,scale_factor:0,scale_factor_limit:0,capacity_per_instance:4" required=2 runner=11Qjxy-Gi subsystem=taskscaler
time="2024-11-13T09:57:26Z" level=info msg="increasing instances" amount=2 group=hetzner/hel1/cpx21/runner-docker-autoscaler runner=11Qjxy-Gi subsystem=taskscaler
time="2024-11-13T09:57:27Z" level=info msg="required scaling change" capacity-info="instance_count:2,max_instance_count:5,acquired:0,unavailable_capacity:0,pending:0,reserved:0,idle_count:8,scale_factor:0,scale_factor_limit:0,capacity_per_instance:4" required=0 runner=11Qjxy-Gi subsystem=taskscaler
time="2024-11-13T09:57:42Z" level=info msg="increasing instances response" group=hetzner/hel1/cpx21/runner-docker-autoscaler num_requested=2 num_successful=2 runner=11Qjxy-Gi subsystem=taskscaler
time="2024-11-13T09:57:42Z" level=info msg="increase update" group=hetzner/hel1/cpx21/runner-docker-autoscaler pending=2 requesting=0 runner=11Qjxy-Gi subsystem=taskscaler total_pending=2
time="2024-11-13T09:57:42Z" level=info msg="instance discovery" cause=requested group=hetzner/hel1/cpx21/runner-docker-autoscaler id="runner-docker-autoscaler-3cfc018b:55575096" runner=11Qjxy-Gi state=running subsystem=taskscaler
time="2024-11-13T09:57:42Z" level=info msg="instance discovery" cause=requested group=hetzner/hel1/cpx21/runner-docker-autoscaler id="runner-docker-autoscaler-dca4e0eb:55575097" runner=11Qjxy-Gi state=running subsystem=taskscaler
```

</details>

We can also follow the logs of the `gitlab-runner` service, and wait for the instances to be ready:

```sh
journalctl --output=cat -f -u gitlab-runner
```

<details><summary>Output</summary>

```
time="2024-11-13T09:57:25Z" level=info msg="plugin initialized" build info="sha=85c314ff; ref=refs/pipelines/1528252336; go=go1.23.2; built_at=2024-11-05T15:20:21+0000; os_arch=linux/amd64" runner=11Qjxy-Gi subsystem=taskscaler version=v0.6.0
time="2024-11-13T09:57:26Z" level=info msg="required scaling change" capacity-info="instance_count:0,max_instance_count:5,acquired:0,unavailable_capacity:0,pending:0,reserved:0,idle_count:8,scale_factor:0,scale_factor_limit:0,capacity_per_instance:4" required=2 runner=11Qjxy-Gi subsystem=taskscaler
time="2024-11-13T09:57:26Z" level=info msg="increasing instances" amount=2 group=hetzner/hel1/cpx21/runner-docker-autoscaler runner=11Qjxy-Gi subsystem=taskscaler
time="2024-11-13T09:57:27Z" level=info msg="required scaling change" capacity-info="instance_count:2,max_instance_count:5,acquired:0,unavailable_capacity:0,pending:0,reserved:0,idle_count:8,scale_factor:0,scale_factor_limit:0,capacity_per_instance:4" required=0 runner=11Qjxy-Gi subsystem=taskscaler
time="2024-11-13T09:57:42Z" level=info msg="increasing instances response" group=hetzner/hel1/cpx21/runner-docker-autoscaler num_requested=2 num_successful=2 runner=11Qjxy-Gi subsystem=taskscaler
time="2024-11-13T09:57:42Z" level=info msg="increase update" group=hetzner/hel1/cpx21/runner-docker-autoscaler pending=2 requesting=0 runner=11Qjxy-Gi subsystem=taskscaler total_pending=2
time="2024-11-13T09:57:42Z" level=info msg="instance discovery" cause=requested group=hetzner/hel1/cpx21/runner-docker-autoscaler id="runner-docker-autoscaler-3cfc018b:55575096" runner=11Qjxy-Gi state=running subsystem=taskscaler
time="2024-11-13T09:57:42Z" level=info msg="instance discovery" cause=requested group=hetzner/hel1/cpx21/runner-docker-autoscaler id="runner-docker-autoscaler-dca4e0eb:55575097" runner=11Qjxy-Gi state=running subsystem=taskscaler
time="2024-11-13T09:58:47Z" level=info msg="instance is ready" instance="runner-docker-autoscaler-3cfc018b:55575096" runner=11Qjxy-Gi subsystem=taskscaler took=1m5.337683491s
time="2024-11-13T09:59:05Z" level=info msg="instance is ready" instance="runner-docker-autoscaler-dca4e0eb:55575097" runner=11Qjxy-Gi subsystem=taskscaler took=1m22.654298839s
```

</details>

We can see that the 2 idle instances are ready after ~1 minute, we can now start running CI pipelines using the new gitlab runner.

To verify, we list all our resources again:

```sh
hcloud all list
```

<details><summary>Output</summary>

```
SERVERS
---
ID         NAME                                STATUS    IPV4              IPV6                      PRIVATE NET   DATACENTER   AGE
55574479   runner-manager                      running   65.109.174.102    2a01:4f9:c012:1a18::/64   -             hel1-dc2     39m
55575096   runner-docker-autoscaler-3cfc018b   running   135.181.24.22     2a01:4f9:c012:f3cf::/64   -             hel1-dc2     17m
55575097   runner-docker-autoscaler-dca4e0eb   running   135.181.107.212   2a01:4f9:c011:ad4c::/64   -             hel1-dc2     17m

PRIMARY IPS
---
ID         TYPE   NAME                  IP                        ASSIGNEE                                   DNS                                             AUTO DELETE   AGE
74302282   ipv4   primary_ip-74302282   65.109.174.102            Server runner-manager                      static.102.174.109.65.clients.your-server.de    yes           39m
74302283   ipv6   primary_ip-74302283   2a01:4f9:c012:1a18::/64   Server runner-manager                      -                                               yes           39m
74303426   ipv4   primary_ip-74303426   135.181.24.22             Server runner-docker-autoscaler-3cfc018b   static.22.24.181.135.clients.your-server.de     yes           17m
74303427   ipv6   primary_ip-74303427   2a01:4f9:c012:f3cf::/64   Server runner-docker-autoscaler-3cfc018b   -                                               yes           17m
74303428   ipv4   primary_ip-74303428   135.181.107.212           Server runner-docker-autoscaler-dca4e0eb   static.212.107.181.135.clients.your-server.de   yes           17m
74303429   ipv6   primary_ip-74303429   2a01:4f9:c011:ad4c::/64   Server runner-docker-autoscaler-dca4e0eb   -                                               yes           17m

FIREWALLS
---
ID        NAME     RULES COUNT   APPLIED TO COUNT
1733905   runner   2 Rules       0 Servers | 1 Label Selector

SSH KEYS
---
ID         NAME                       FINGERPRINT                                       AGE
22499499   dev                        2b:9f:a0:6d:01:12:a4:4d:2b:27:02:34:56:bf:fe:5f   45m
24523700   runner-docker-autoscaler   6a:bc:f8:da:df:0f:5c:19:aa:20:93:48:e5:13:38:40   17m
```

</details>

## 3. Next steps

We have configured a basic GitLab CI infrastructure, the next steps are to:

- [Configure a shared cache](../guides/shared-cache.md)
- [Configure monitoring](../guides/monitoring.md)
- If needed, [configure volumes](../guides/volumes.md)