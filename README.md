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

## Setting an IAM policy for the runner

### Our recommendations:


- Grant least privilege
- Create an IAM group with a policy like the example below and assign each AWS runner user to the group
- Use policy conditions for extra security. This will depend on your setup.
- Do not share AWS access keys among runners. One runner = one user = one access key

Create the runner's User with an `AWS Credential Type` of `Access key - Programmatic access`, enabling the runner to 
access your ASG via the AWS SDK.

#### Recommended IAM Policy:

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
            "Action": "ec2-instance-connect:SendSSHPublicKey",
            "Resource": "*",
            "Condition": {
                "StringEquals": {
                    "ec2:ResourceTag/aws:autoscaling:groupName": "YOUR_AUTOSCALING_GROUP_NAME"
                }
            }
        }
    ]
}
```
