# Attach volumes

If your instance storage requirements are not met by the server type you need, you can attach volumes to your instances to increase the storage capacity. This document describe the steps to attach volumes to your instances using the Hetzner Fleeting plugin.

## Configure the desired volume size

First, you must define the size of the volume you want to attach to your instances:

```diff
 // ...
 [runners.autoscaler.plugin_config]
 name = "runner-docker-autoscaler0"
 token = "<your-hetzner-cloud-token>"

 location = "fsn1"
 server_type = "cpx41"
 image = "debian-12"
+volume_size = 200

 user_data = """#cloud-config
 package_update: true
 package_upgrade: true

 apt:
   sources:
     docker.list:
       source: deb https://download.docker.com/linux/debian $RELEASE stable
       keyid: 9DC858229FC7DD38854AE2D88D81803C0EBFCD88

 packages:
   - docker-ce
 """
```

For more details about the `volume_size` config, see the [plugin configuration reference](../references/configuration.md#plugin-configuration).

## Format and mount the volume

With the volume attached to the instance, you must now format and mount the volume in your operating system. This can be accomplished using `cloud-init`:

```diff
 // ...
 [runners.autoscaler.plugin_config]
 name = "runner-docker-autoscaler0"
 token = "<your-hetzner-cloud-token>"

 location = "fsn1"
 server_type = "cpx41"
 image = "debian-12"
 volume_size = 200

 user_data = """#cloud-config
 package_update: true
 package_upgrade: true

 apt:
   sources:
     docker.list:
       source: deb https://download.docker.com/linux/debian $RELEASE stable
       keyid: 9DC858229FC7DD38854AE2D88D81803C0EBFCD88

 packages:
   - docker-ce

+bootcmd:
+  - mkfs.ext4 -F -m 0 /dev/disk/by-id/scsi-SHC_Volume_*
+  - mount /dev/disk/by-id/scsi-SHC_Volume_* /mnt
 """
```

> Note that the above commands assume you have a single volume attached to your server.

## Use the volume

The additional storage capacity can now be used. Below are some examples how to:

- [Configure directories for the container build and cache ](https://docs.gitlab.com/runner/executors/docker.html#configure-directories-for-the-container-build-and-cache) and [Mount a host directory as a data volume](https://docs.gitlab.com/runner/configuration/advanced-configuration.html#example-2-mount-a-host-directory-as-a-data-volume).
