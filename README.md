# Fleeting Plugin Hetzner

This is a [fleeting](https://gitlab.com/gitlab-org/fleeting/fleeting) plugin for Hetzner.

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

**TODO**: go through the rest of this file. No content below this line has been modified.

## Plugin Configuration

The following parameters are supported:

| Parameter             | Type   | Description |
|-----------------------|--------|-------------|
| `name`                | string | Name of the Auto Scaling Group |
| `profile` | string | Optional. AWS profile-name ([Named profiles for the AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html)). |
| `config_file`    | string | Optional. Path to the AWS config file ([AWS Configuration and credential file settings](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html)). |
| `credentials_file`    | string | Optional. Path to the AWS credential file ([AWS Configuration and credential file settings](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html)). |

The credentials don't need to be set if the plugin is running on an instance inside AWS
with the IAM permission assigned. See [Recommended IAM Policy](#recommended-iam-policy)

### Default connector config

| Parameter                | Default  |
|--------------------------|----------|
| `os`                     | `linux`  |
| `protocol`               | `ssh` or `winrm` if Windows OS is detected |
| `username`               | `ec2-user` or `Administrator` if Windows OS is detected |
| `use_static_credentials` | `false`  |
| `key_path`               | None. This is the path for the private key file used to connect to the runner **manager** machine. Required for Windows OS. |

For Windows instances, if `use_static_credentials` is false, the password field is populated with a password that AWS provisions.

For other instances, if `use_static_credentials` is false, credentials will be set using [SendSSHPublicKey](https://docs.aws.amazon.com/ec2-instance-connect/latest/APIReference/API_SendSSHPublicKey.html), either using the specified key or dynamically creating one.

## Autoscaling Group Setup

- Group size desired and minimal capacity should be zero.
- Maximum capacity should be equal or more than the configured fleeting Max Size option.
- Scaling policy should be set to `None`.
- Process `AZRebalance` should be suspended.
- Instance scale-in protection should be enabled.

## Setting an IAM policy

### Our recommendations

- Grant least privilege
- Create an IAM group with a policy like the example below and assign each AWS user to the group
- Use policy conditions for extra security. This will depend on your setup.
- Do not share AWS access keys to separate deployments.

Create an `AWS Credential Type` of `Access key - Programmatic access`, enabling the plugin to
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
                "autoscaling:DescribeAutoScalingGroups",
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
is set to `false` (default).

The IAM policy for `ec2:GetPasswordData` is only necessary if the EC2 instances runs on Windows.

## Examples

### GitLab Runner

GitLab Runner has examples on using this plugin for the [Instance executor](https://docs.gitlab.com/runner/executors/instance.html#examples) and [Docker Autoscaler executor](https://docs.gitlab.com/runner/executors/docker_autoscaler.html#examples).
