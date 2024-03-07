package fake

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	asgtypes "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2instanceconnect"
)

type Instance struct {
	InstanceId string
	State      asgtypes.LifecycleState
}

type Client struct {
	cfg aws.Config

	Name            string
	Instances       []Instance
	DesiredCapacity int

	count atomic.Uint64
}

var once sync.Once
var winrmKey *rsa.PrivateKey

func Key() *rsa.PrivateKey {
	once.Do(func() {
		var err error
		winrmKey, err = rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			panic(err)
		}
	})

	return winrmKey
}

func New(cfg aws.Config) *Client {
	return &Client{cfg: cfg}
}

func (c *Client) SetDesiredCapacity(ctx context.Context, params *autoscaling.SetDesiredCapacityInput, optFns ...func(*autoscaling.Options)) (*autoscaling.SetDesiredCapacityOutput, error) {
	c.DesiredCapacity = int(aws.ToInt32(params.DesiredCapacity))

	for len(c.Instances) < c.DesiredCapacity {
		c.Instances = append(c.Instances, Instance{
			InstanceId: fmt.Sprintf("instance__%d", c.count.Add(1)),
			State:      asgtypes.LifecycleStateInService,
		})
	}

	return &autoscaling.SetDesiredCapacityOutput{}, nil
}

func (c *Client) TerminateInstanceInAutoScalingGroup(ctx context.Context, params *autoscaling.TerminateInstanceInAutoScalingGroupInput, optFns ...func(*autoscaling.Options)) (*autoscaling.TerminateInstanceInAutoScalingGroupOutput, error) {
	for idx, instance := range c.Instances {
		if instance.InstanceId == aws.ToString(params.InstanceId) {
			c.Instances[idx].State = asgtypes.LifecycleStateTerminated
		}
	}

	return &autoscaling.TerminateInstanceInAutoScalingGroupOutput{}, nil
}

func (c *Client) DescribeAutoScalingGroups(ctx context.Context, params *autoscaling.DescribeAutoScalingGroupsInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
	var instances []asgtypes.Instance

	for idx, instance := range c.Instances {
		if instance.State == asgtypes.LifecycleStateTerminated {
			c.Instances = append(c.Instances[:idx], c.Instances[idx+1:]...)
			c.DesiredCapacity--
		}
	}

	for _, instance := range c.Instances {
		instances = append(instances, asgtypes.Instance{
			InstanceId:     &instance.InstanceId,
			InstanceType:   aws.String(string(ec2types.InstanceTypeC3Large)),
			LifecycleState: instance.State,
		})
	}

	return &autoscaling.DescribeAutoScalingGroupsOutput{
		AutoScalingGroups: []asgtypes.AutoScalingGroup{
			{
				AutoScalingGroupName:             &params.AutoScalingGroupNames[0],
				DesiredCapacity:                  aws.Int32(int32(c.DesiredCapacity)),
				Instances:                        instances,
				NewInstancesProtectedFromScaleIn: aws.Bool(true),
			},
		},
	}, nil
}

func (c *Client) DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	var instances []ec2types.Instance

	for _, instance := range c.Instances {
		instances = append(instances, ec2types.Instance{
			InstanceId:   &instance.InstanceId,
			InstanceType: ec2types.InstanceTypeC3Large,
			Placement: &ec2types.Placement{
				AvailabilityZone: aws.String("us-west-2"),
			},
		})
	}

	return &ec2.DescribeInstancesOutput{Reservations: []ec2types.Reservation{{Instances: instances}}}, nil
}

func (c *Client) GetPasswordData(ctx context.Context, params *ec2.GetPasswordDataInput, optFns ...func(*ec2.Options)) (*ec2.GetPasswordDataOutput, error) {
	encoded, err := rsa.EncryptPKCS1v15(rand.Reader, &Key().PublicKey, []byte("password"))
	if err != nil {
		return nil, err
	}

	return &ec2.GetPasswordDataOutput{
		PasswordData: aws.String(base64.StdEncoding.EncodeToString(encoded)),
	}, nil
}

func (c *Client) SendSSHPublicKey(ctx context.Context, params *ec2instanceconnect.SendSSHPublicKeyInput, optFns ...func(*ec2instanceconnect.Options)) (*ec2instanceconnect.SendSSHPublicKeyOutput, error) {
	return &ec2instanceconnect.SendSSHPublicKeyOutput{
		Success: true,
	}, nil
}
