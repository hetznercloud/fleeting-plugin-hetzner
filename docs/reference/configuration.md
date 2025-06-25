# Configuration reference

This page references the different configurations for the Hetzner Cloud fleeting plugin.

[TOC]

## Plugin configuration

The [`[runners.autoscaler.plugin_config]` section](https://docs.gitlab.com/runner/configuration/advanced-configuration.html#the-runnersautoscalerplugin_config-section) supports the following parameters:

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
      Name of the fleeting plugin instance group. The created instance names will be
      prefixed using this name.
    </td>
  </tr>
  <tr>
    <td><code>token</code></td>
    <td>string (<strong>required</strong>)</td>
    <td>
      <a href="https://docs.hetzner.com/cloud/api/getting-started/generating-api-token">Hetzner Cloud API token</a>
      to access your Hetzner Cloud project.
      <br>
      You may also use the <code>HCLOUD_TOKEN</code> or <code>HCLOUD_TOKEN_FILE</code> environment variable
      to configure the token.
    </td>
  </tr>
  <tr>
    <td><code>endpoint</code></td>
    <td>string</td>
    <td>
      Hetzner Cloud API endpoint to use.
      <br>
      You may also use the <code>HCLOUD_ENDPOINT</code> or <code>HCLOUD_ENDPOINT_FILE</code> environment variable
      to configure the endpoint.
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
    <td>string or list of string (<strong>required</strong>)</td>
    <td>
      <a href="https://docs.hetzner.com/cloud/servers/overview/">Hetzner Cloud server type</a>
      on which the instances will run. Using a list of server types allows you to define
      additional server types to fallback to in case of unavailable resource errors. All
      servers types must have the same CPU architecture.
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
      Disable the instances public IPv4/IPv6. If no public IPs are enabled, you must
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
      [Label selector](https://docs.hetzner.cloud/reference/cloud#label-selector) used to filter the
      Hetzner Cloud Primary IPs in your Hetzner Cloud project when populating the public
      IP pool.
    </td>
  </tr>
  <tr>
    <td><code>private_networks</code></td>
    <td>list of string</td>
    <td>
      List of Hetzner Cloud Networks the instances will be attached to. To communicate
      with the instances via the private network, you must configure the connector to
      use the internal address (see the connector <code>use_external_addr</code> config).
    </td>
  </tr>
  <tr>
    <td><code>user_data</code> and <code>user_data_file</code></td>
    <td>string</td>
    <td>
      Configuration for the provisioning utility that runs during the instances creation.
      On Ubuntu, you can provide a Cloud Init configuration to setup the instances. Make
      sure to wait for the instances to be ready before scheduling jobs on them by using
      the autoscaler <code>instance_ready_command</code> config.
      Note that <code>user_data</code> and <code>user_data_file</code> are mutually exclusive.
    </td>
  </tr>
  <tr>
    <td><code>volume_size</code></td>
    <td>integer</td>
    <td>
      Size in GB for the <a href="https://docs.hetzner.com/cloud/volumes/overview">Volume</a>
      that will be attached to each instance. No Volume will be attached if the
      <code>volume_size</code> is 0 GB. The minimal <code>volume_size</code> is 10 GB.
    </td>
  </tr>
  <tr>
    <td><code>labels</code></td>
    <td>map of string</td>
    <td>
      User-defined <a href="https://docs.hetzner.cloud/reference/cloud#labels">labels</a> (key/value pairs)
      that will be set on the instances.
    </td>
  </tr>
</table>

## Autoscaler configuration

Below are parameters from the [`[runners.autoscaler]` section](https://docs.gitlab.com/runner/configuration/advanced-configuration.html#the-runnersautoscaler-section) that are important for our plugin:

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

Below are parameters from the [`[runners.autoscaler.connector_config]` section](https://docs.gitlab.com/runner/configuration/advanced-configuration.html#the-runnersautoscalerconnector_config-section) that are important for our plugin:

<table>
  <tr>
    <th>Parameter</th>
    <th>Value</th>
  </tr>
  <tr>
    <td><code>use_external_addr</code></td>
    <td>
      Access the instances through their public addresses. Note that without private
      networks, this field must be set to <code>true</code>.
    </td>
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
