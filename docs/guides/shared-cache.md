# Enable shared cache using the Hetzner Object Storage

A local cache provides limited caching capabilities for your CI jobs running across many instances. This document describes the steps to enable a shared cache using the [Hetzner Object Storage](https://docs.hetzner.com/storage/object-storage) and the Hetzner fleeting plugin.

## Creating S3 credentials and bucket

First, you must generate S3 credentials using the ["Generating S3 keys" guide](https://docs.hetzner.com/storage/object-storage/getting-started/generating-s3-keys/).

Once you have the credentials, you must create a new S3 Bucket using the ["Creating a Bucket" guide](https://docs.hetzner.com/storage/object-storage/getting-started/creating-a-bucket/).

> It is recommended to add a random suffix to the name of your Bucket, e.g. `gitlab-ci-cache-7d2f6722`.

## Configuring the `gitlab-runner`

Now that you gathered all the required data, you can configure the `gitlab-runner` to enable the S3 shared cache:

```toml
[runners.cache]
Type = "s3"
Shared = true

[runners.cache.s3]
ServerAddress = "fsn1.your-objectstorage.com"
BucketName = "gitlab-ci-cache-7d2f6722"
AccessKey = "LR3IWEWNCFHV46133962"
SecretKey = "URyhbHBO6bMAoNmRJTf0TmTeXCngCMBGnjbvIp9g"
```

> Make sure to update the:
>
> - `ServerAddress` with the location you chose.
> - `BucketName` with the Bucket name you chose.
> - `AccessKey` with the access key you generated.
> - `SecretKey` with the secret key you generated.

For more details about the `[runners.cache]` config, see the [`gitlab-runner` cache configuration reference](https://docs.gitlab.com/runner/configuration/advanced-configuration.html#the-runnerscache-section).
