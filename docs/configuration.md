# Configuration reference

This page references the different configuration for the Hetzner Cloud fleeting plugin.

[TOC]

## Plugin configuration

The [`[runners.autoscaler.plugin_config]` section](https://docs.gitlab.com/runner/configuration/advanced-configuration.html#the-runnersautoscalerplugin_config-section) support the following parameters:

<table>
  <tr>
    <th>Parameter</th>
    <th>Type</th>
    <th>Description</th>
  </tr>
  <tr>
    <td><code>name</code></td>
    <td>string (<strong>required</strong>)</td>
    <td>
      Name of the fleeting plugin instance group. The created instances name will be
      prefixed using this name.
    </td>
  </tr>
  <tr>
    <td><code>token</code></td>
    <td>string (<strong>required</strong>)</td>
    <td>
      <a href="https://docs.hetzner.com/cloud/api/getting-started/generating-api-token">Hetzner Cloud API token</a>
      to access your Hetzner Cloud Project.
    </td>
  </tr>
  <tr>
    <td><code>endpoint</code></td>
    <td>string</td>
    <td>
      Hetzner Cloud API endpoint to use.
    </td>
  </tr>
  <tr>
    <td><code>location</code></td>
    <td>string (<strong>required</strong>)</td>
    <td>
      <a href="https://docs.hetzner.com/cloud/general/locations/">Hetzner Cloud location</a>
      in which the instances will run.
      <br>
      You can list the available locations by running <code>hcloud location list</code>.
    </td>
  </tr>
  <tr>
    <td><code>server_type</code></td>
    <td>string (<strong>required</strong>)</td>
    <td>
      <a href="https://docs.hetzner.com/cloud/servers/overview/">Hetzner Cloud server type</a>
      on which the instances will run.
      <br>
      You can list the available server types by running <code>hcloud server-type list</code>.
    </td>
  </tr>
  <tr>
    <td><code>image</code></td>
    <td>string (<strong>required</strong>)</td>
    <td>
      Hetzner Cloud image from which the instances will run.
      <br>
      You can list the available images by running <code>hcloud image list</code>.
    </td>
  </tr>
  <tr>
    <td><code>public_ipv4_disabled</code> and <code>public_ipv6_disabled</code></td>
    <td>boolean</td>
    <td>
      Disable the instances public ipv4/ipv6. If no public IPs are enabled, you must
      enable a private network (see the <code>private_networks</code> config) to be able
      to communicate with the instances.
    </td>
  </tr>
  <tr>
    <td><code>public_ip_pool_enabled</code></td>
    <td>boolean</td>
    <td>
      Enable a public IP pool, from which Hetzner Cloud Primary IPs will be picked when
      creating new instances. This feature offers a way to have predictable public IPs
      for the fleeting instances.
    </td>
  </tr>
  <tr>
    <td><code>public_ip_pool_selector</code></td>
    <td>string</td>
    <td>
      Label selector (https://docs.hetzner.cloud/#label-selector) used to filter the
      Hetzner Cloud Primary IPs in your Hetzner Cloud Project when populating the public
      IP pool.
    </td>
  </tr>
  <tr>
    <td><code>private_networks</code></td>
    <td>list of string</td>
    <td>
      List of Hetzner Cloud networks the instances will be attached to. To communicate
      with the instances via the private network, you must configure the connector to
      use the internal address (see the connector <code>use_external_addr</code> config).
    </td>
  </tr>
  <tr>
    <td><code>user_data</code> and <code>user_data_file</code></td>
    <td>string</td>
    <td>
      Configuration for the provisioning utility that run during the instances creation.
      On Ubuntu, you can provide a Cloud Init configuration to setup the instances. Make
      sure to wait for the instances to be ready before scheduling jobs on them by using
      the autoscaler <code>instance_ready_command</code> config.
      Note that <code>user_data</code> and <code>user_data_file</code> are mutually exclusive.
    </td>
  </tr>
</table>

## Autoscaler configuration

The [`[runners.autoscaler]` section](https://docs.gitlab.com/runner/configuration/advanced-configuration.html#the-runnersautoscaler-section) have parameters that may interest you:

<table>
  <tr>
    <th>Parameter</th>
    <th>Description</th>
  </tr>
  <tr>
    <td><code>instance_ready_command</code></td>
    <td>
      When using the <code>user_data</code> or <code>user_data_file</code> config, you
      must wait for the instances to be ready before scheduling jobs on them. When using
      Cloud Init, this can be done with the following: <code>cloud-init status --wait || test $? -eq 2</code>
    </td>
  </tr>
</table>

## Connector configuration

The [`[runners.autoscaler.connector_config]` section](https://docs.gitlab.com/runner/configuration/advanced-configuration.html#the-runnersautoscalerconnector_config-section) have parameters that are only partially supported:

<table>
  <tr>
    <th>Parameter</th>
    <th>Value</th>
  </tr>
  <tr>
    <td><code>os</code></td>
    <td>Only <code>linux</code> is supported.</td>
  </tr>
    <tr>
    <td><code>protocol</code></td>
    <td>Only <code>ssh</code> is supported.</td>
  </tr>
</table>
