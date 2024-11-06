# Enable shared cache using the Hetzner Object Storage

A local cache provides limited caching capabilities for your CI jobs running across many instances. This document describe the steps to enable a shared cache using the [Hetzner Object Storage](https://docs.hetzner.com/storage/object-storage) and the Hetzner Fleeting plugin.

## Creating s3 credentials and bucket

First, you must generate s3 credentials using the [Generating S3 keys guide](https://docs.hetzner.com/storage/object-storage/getting-started/generating-s3-keys/).

Once you have the credentials, you must create a new s3 bucket using the [Creating a Bucket guide](https://docs.hetzner.com/storage/object-storage/getting-started/creating-a-bucket/).

> It is recommended to add a random suffix to the name of your bucket, e.g. `gitlab-ci-cache-7d2f6722`.

## Configuring the `gitlab-runner`

Now that you gathered all the required data, you can configure the `gitlab-runner` to enable the s3 shared cache:

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
> - `ServerAddress` with the location you choose.
> - `BucketName` with the bucket name you choose.
> - `AccessKey` with the access key you generated.
> - `SecretKey` with the secret key you generated.

For more details about the `[runners.cache]` config, see the [`gitlab-runner` cache configuration reference](https://docs.gitlab.com/runner/configuration/advanced-configuration.html#the-runnerscache-section).
