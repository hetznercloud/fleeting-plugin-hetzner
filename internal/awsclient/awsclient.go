package awsclient

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2instanceconnect"
)

type Client interface {
	SetDesiredCapacity(context.Context, *autoscaling.SetDesiredCapacityInput, ...func(*autoscaling.Options)) (*autoscaling.SetDesiredCapacityOutput, error)
	TerminateInstanceInAutoScalingGroup(context.Context, *autoscaling.TerminateInstanceInAutoScalingGroupInput, ...func(*autoscaling.Options)) (*autoscaling.TerminateInstanceInAutoScalingGroupOutput, error)
	DescribeAutoScalingGroups(context.Context, *autoscaling.DescribeAutoScalingGroupsInput, ...func(*autoscaling.Options)) (*autoscaling.DescribeAutoScalingGroupsOutput, error)
	DescribeInstances(context.Context, *ec2.DescribeInstancesInput, ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	GetPasswordData(context.Context, *ec2.GetPasswordDataInput, ...func(*ec2.Options)) (*ec2.GetPasswordDataOutput, error)
	SendSSHPublicKey(context.Context, *ec2instanceconnect.SendSSHPublicKeyInput, ...func(*ec2instanceconnect.Options)) (*ec2instanceconnect.SendSSHPublicKeyOutput, error)
}

var _ Client = (*client)(nil)

type client struct {
	autoscaler *autoscaling.Client
	ec2        *ec2.Client
	ec2connect *ec2instanceconnect.Client
}

func New(cfg aws.Config) Client {
	return &client{
		autoscaler: autoscaling.NewFromConfig(cfg),
		ec2:        ec2.NewFromConfig(cfg),
		ec2connect: ec2instanceconnect.NewFromConfig(cfg),
	}
}

func (c *client) SetDesiredCapacity(ctx context.Context, params *autoscaling.SetDesiredCapacityInput, optFns ...func(*autoscaling.Options)) (*autoscaling.SetDesiredCapacityOutput, error) {
	return c.autoscaler.SetDesiredCapacity(ctx, params, optFns...)
}

func (c *client) TerminateInstanceInAutoScalingGroup(ctx context.Context, params *autoscaling.TerminateInstanceInAutoScalingGroupInput, optFns ...func(*autoscaling.Options)) (*autoscaling.TerminateInstanceInAutoScalingGroupOutput, error) {
	return c.autoscaler.TerminateInstanceInAutoScalingGroup(ctx, params, optFns...)
}

func (c *client) DescribeAutoScalingGroups(ctx context.Context, params *autoscaling.DescribeAutoScalingGroupsInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
	return c.autoscaler.DescribeAutoScalingGroups(ctx, params, optFns...)
}

func (c *client) DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	return c.ec2.DescribeInstances(ctx, params, optFns...)
}

func (c *client) GetPasswordData(ctx context.Context, params *ec2.GetPasswordDataInput, optFns ...func(*ec2.Options)) (*ec2.GetPasswordDataOutput, error) {
	return c.ec2.GetPasswordData(ctx, params, optFns...)
}

func (c *client) SendSSHPublicKey(ctx context.Context, params *ec2instanceconnect.SendSSHPublicKeyInput, optFns ...func(*ec2instanceconnect.Options)) (*ec2instanceconnect.SendSSHPublicKeyOutput, error) {
	return c.ec2connect.SendSSHPublicKey(ctx, params, optFns...)
}
