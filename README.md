# Fleeting Plugin AWS

This is a go plugin for fleeting on AWS. It is intended to be run by GitLab Runner, and cannot be run directly.

[![Pipeline Status](https://gitlab.com/gitlab-org/fleeting/fleeting-plugin-aws/badges/main/pipeline.svg)](https://gitlab.com/gitlab-org/fleeting/fleeting-plugin-aws/commits/main)
[![Go Report Card](https://goreportcard.com/badge/gitlab.com/gitlab-org/fleeting/fleeting-plugin-aws)](https://goreportcard.com/report/gitlab.com/gitlab-org/fleeting/fleeting-plugin-aws)

## Building the plugin

To run Gitlab Runner with this plugin, generate an executable binary and place it on your system's PATH.

To generate the binary, ensure `$GOPATH/bin` is on your PATH, then use `go install`:

```shell
cd cmd/fleeting-plugin-aws/
go install 
```

If you are managing go versions with asdf, run this after generating the binary:

```shell
asdf reshim
```

## Plugin Configuration

The configuration of the plugin is part of the runner configuration in the usual way using `config.toml`.

### Example

This example illustrates the parameters set in the `config.toml` file for the plugin:

```toml
[runners.autoscaler.plugin_config]
      credentials_file = "/home/user/.aws/credentials"
      name = "gitlab-taskrunner-asg"
      region = "us-east-1"
[runners.autoscaler.connector_config]
      username = "ubuntu"
```

### The `[runners.autoscaler.plugin_config]` section

The following parameters configure the plugin for fleeting on AWS.

| Parameter             | Type   | Description |
|-----------------------|--------|-------------|
| `credentials_profile` | string | Optional. AWS profile-name ([Named profiles for the AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html)). Conflicts with `credentials_file`. |
| `credentials_file`    | string | Optional. Path to the AWS credential file ([AWS Configuration and credential file settings](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html)). Conflicts with `credentials_profile`. |
| `name`                | string | Name of the Auto Scaling Group |
| `region`              | string | Name of the region of the Auto Scaling Group |

The credentials don't needed to be set if the runner is on AWS and the runner instance
has the IAM permission assigned. See [Recommended IAM Policy](#recommended-iam-policy)

### The `[runners.autoscaler.connector_config]` section

The following parameters configure the connector to the AWS EC2 instance

| Parameter                | Type   | Description |
|--------------------------|--------|-------------|
| `os`                     | string | Optional. Possible Values: `linux` (Default), `windows` or `darwin`. |
| `arch`                   | string | Optional. Possible Values: `x86_64` or `arm64` |
| `protocol`               | string | Optional. Possible Values: `ssh` or `winrm` |
| `username`               | string | Optional. Default: `ec2-user`. The user will have access to the instance.|
| `password`               | string | Optional. |
| `key_path`               | string | Path to the private key file. This parameter is required for Windows instances |
| `use_static_credentials` | bool   | Optional. Default: `false` |
| `keepalive`              | int64 | Optional. |
| `timeout`                | int64 | Optional. |

The connector detects `os`, `arch` and `protocol` based on the information of the instance provided by AWS API.
The plugin uses a dynamically created key to connected to the instance. You need to set the username if the login
user is not `ec2-user`, e.g. in case of Ubuntu it is `ubuntu`. `Administrator` will be assumed for Windows instances as `username` if not set.
You need to set `password` or `key_path` if `use_static_credentials` is set to true for non Windows instances.

## Setting an IAM policy for the runner

### Our recommendations

- Grant least privilege
- Create an IAM group with a policy like the example below and assign each AWS runner user to the group
- Use policy conditions for extra security. This will depend on your setup.
- Do not share AWS access keys among runners. One runner = one user = one access key

Create the runner's user with an `AWS Credential Type` of `Access key - Programmatic access`, enabling the runner to
access your ASG via the AWS SDK.

#### Recommended IAM Policy

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "autoscaling:SetDesiredCapacity",
                "autoscaling:TerminateInstanceInAutoScalingGroup"
            ],
            "Resource": "YOUR_AUTOSCALING_GROUP_ARN"
        },
        {
            "Effect": "Allow",
            "Action": [
                "autoscaling:DescribeAutoScalingInstances",
                "ec2:DescribeInstances"
            ],
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "ec2:GetPasswordData",
                "ec2-instance-connect:SendSSHPublicKey"
            ],
            "Resource": "arn:aws:ec2:YOUR_AWS_REGION:YOUR_AWS_ACCOUNT_ID:instance/*",
            "Condition": {
                "StringEquals": {
                    "ec2:ResourceTag/aws:autoscaling:groupName": "YOUR_AUTOSCALING_GROUP_NAME"
                }
            }
        }
    ]
}
```

The IAM policy for `ec2-instance-connect:SendSSHPublicKey` is only necessary if the configuration `use_static_credentials`
is set to `true` (default).

The IAM policy for `ec2:GetPasswordData` is only necessary if the EC2 instances runs on Windows.

## WinRM

Gitlab Runner does use Basic authentication via WinRM-HTTP (TCP/5985) to connect to the EC2 instance.
The Windows AMIs provided by AWS doesn't allow WinRM access by default.

The Windows AMI for the EC2 instance shall be adjusted to be used with Gitlab Runner.
The Windows firewall shall be open for WinRM-HTTP (TCP/5985). The WinRM service shall be
configured to allow Basic authentication via an unencrypted connection (WinRM-HTTP).

```powershell
netsh advfirewall firewall add rule name="WinRM-HTTP" dir=in localport=5985 protocol=TCP action=allow
winrm set winrm/config/service/auth '@{Basic="true"}'
winrm set winrm/config/service '@{AllowUnencrypted="true"}'
```
