package aws

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type Vpc struct {
	Client *ec2.Client
}

func NewVpc(awsClient *ec2.Client) (*Vpc, error) {
	return &Vpc{Client: awsClient}, nil
}


type VpcInfo struct {
	VpcId     *string  `json:"VpcId"`
	VpcName   *string  `json:"VpcName"`
	CidrBlock *string  `json:"CidrBlock"`
}

type VpcData struct {
	TotalCount int        `json:"TotalCount"`
	Vpcs       []*VpcInfo `json:"Vpcs"`
}

func (v *Vpc)ListVpcs(pageSize, pageNumber string) (interface{}, error) {
	input := &ec2.DescribeVpcsInput{}
	vpcData := VpcData{Vpcs: []*VpcInfo{}}
	req, err := v.Client.DescribeVpcs(context.TODO(), input)
	if err == nil {
		vpcData.TotalCount = len(req.Vpcs)
		for _, vpc := range req.Vpcs {
			var name *string
			for _, tag := range vpc.Tags {
				if tag.Key != nil && *tag.Key == "Name" {
					name = tag.Value
					break
				}
			}
			vpcData.Vpcs = append(vpcData.Vpcs , &VpcInfo{
				VpcId: vpc.VpcId,
				CidrBlock: vpc.CidrBlock,
				VpcName: name,
			})
		}
	}
	return vpcData, err
}

type Subnet struct {
	VpcId                   *string  `json:"VpcId"`
	SubnetName              *string  `json:"SubnetName"`
	SubnetId                *string  `json:"SubnetId"`
	CidrBlock               *string  `json:"CidrBlock"`
	AvailabilityZone        *string  `json:"AvailabilityZone"`
	State                   string   `json:"State"`
	AvailableIpAddressCount int32    `json:"AvailableIpAddressCount"`
}

type SubnetData struct {
	TotalCount int        `json:"TotalCount"`
	Subnets    []*Subnet  `json:"Subnets"`
}

func (v *Vpc) ListSubnets(vpcId, pageSize, pageNumber string) (interface{}, error) {
	/* 过滤值参考命令行输出， 全部小写，多单词加"-" */
	fmt.Println("=========")
	input := &ec2.DescribeSubnetsInput{
		Filters: []types.Filter{
			{
				Name: aws.String("vpc-id"),
				Values: []string{
					vpcId,
				},
			},
			{
				Name: aws.String("tag:cloud-manager"),
				Values: []string{
					"true",
				},
			},
		},
	}
	snData := SubnetData{Subnets: []*Subnet{}}
	req, err := v.Client.DescribeSubnets(context.TODO(), input)
	if err == nil {
		snData.TotalCount = len(req.Subnets)
		for _, sn := range req.Subnets {
			var name *string
			for _, tag := range sn.Tags {
				if tag.Key != nil && *tag.Key == "Name" {
					name = tag.Value
					break
				}
			}
			snData.Subnets = append(snData.Subnets, &Subnet{
				VpcId: sn.VpcId,
				SubnetId: sn.SubnetId,
				SubnetName: name,
				CidrBlock: sn.CidrBlock,
				State: string(sn.State),
				AvailabilityZone: sn.AvailabilityZone,
				AvailableIpAddressCount: sn.AvailableIpAddressCount,
			})
		}
	}
	return snData, err
}


type  EipAddress struct {
	AllocationId       *string  `json:"Id"`
	Name               *string  `json:"Name"`
	PublicIp           *string  `json:"PublicIp"`
	PrivateIp          *string  `json:"PrivateIp"`
	InstanceId         *string  `json:"InstanceId"`
	NetworkInterfaceId *string  `json:"NetworkInterfaceId"`
	InUse              bool     `json:"InUse"`

}

type EipData struct {
	TotalCount   int           `json:"TotalCount"`
	EipAddresses []*EipAddress `json:"EipAddresses"`
}
func (v *Vpc)ListPublicIpv4s(pageSize, pageNumber string) (interface{}, error) {
	input := &ec2.DescribeAddressesInput{}
	eipData := EipData{EipAddresses: []*EipAddress{}}
	req, err := v.Client.DescribeAddresses(context.TODO(), input)
	if err == nil {
		for _, eip := range req.Addresses {
			var name *string
			for _, tag := range eip.Tags {
				if tag.Key != nil && *tag.Key == "Name" {
					name = tag.Value
					break
				}
			}
			//过滤掉已使用的
			var inUse bool
			if eip.InstanceId != nil {
				inUse = true
				continue
			}
			eipData.EipAddresses = append(eipData.EipAddresses, &EipAddress{
				AllocationId: eip.AllocationId,
				Name: name,
				PublicIp: eip.PublicIp,
				PrivateIp: eip.PrivateIpAddress,
				InstanceId: eip.InstanceId,
				NetworkInterfaceId: eip.NetworkInterfaceId,
				InUse: inUse,
			})
		}
	}
	eipData.TotalCount = len(eipData.EipAddresses)
	return eipData, err
}