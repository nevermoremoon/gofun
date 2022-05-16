package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
)

type EKS struct {
	Client *eks.Client
}

func NewEKS(awsClient *eks.Client) (*EKS, error) {
	return &EKS{Client: awsClient}, nil
}


type NodeGroup struct {
	Cluster string    `json:"cluster"`
	Name string       `json:"name"`
	MaxSize int32     `json:"maxSiz"`
	MinSize int32     `json:"minSize"`
	DesiredSize int32 `json:"desiredSize"`
}

func (e *EKS) getNodeGroup(clusterName, groupName string) (nodegroup *NodeGroup, err error){
	input := &eks.DescribeNodegroupInput{
		ClusterName: aws.String(clusterName),
		NodegroupName: aws.String(groupName),
	}

	req, err := e.Client.DescribeNodegroup(context.TODO(), input)
	if err != nil {
		err = fmt.Errorf("eks %s get nodegroup %s err: %s",clusterName, groupName, err.Error())
		return
	}

	if req.Nodegroup != nil && req.Nodegroup.ScalingConfig != nil {
		nodegroup = &NodeGroup {
			Cluster:     clusterName,
			Name:        groupName,
			MaxSize:     aws.ToInt32(req.Nodegroup.ScalingConfig.MaxSize),
			MinSize:     aws.ToInt32(req.Nodegroup.ScalingConfig.MinSize),
			DesiredSize: aws.ToInt32(req.Nodegroup.ScalingConfig.DesiredSize),
		}
	} else {
		err = fmt.Errorf("eks %s get nodegroup %s err: %s", clusterName, groupName, "nodegroup or size not found")
	}
	return
}

func (e *EKS) NodeGroupScale(cluster, group, service string, number int32) (err error) {
	nodegroup, err := e.getNodeGroup(cluster, group)
	if err != nil {
		return
	}

	size := nodegroup.DesiredSize + number
	if nodegroup.DesiredSize == 0 {
		err = fmt.Errorf("eks %s nodegroup %s current size is 0, forbbiden scale up", cluster, group)
	} else if size > nodegroup.MaxSize {
		err = fmt.Errorf("eks %s nodegroup %s target size is %d, above max size %d, forbbiden scale up", cluster, group, size, nodegroup.MaxSize)
	}

	if err != nil {
		return
	}

	input := &eks.UpdateNodegroupConfigInput{
		ClusterName:        aws.String(cluster),
		NodegroupName:      aws.String(group),
		Labels:             &types.UpdateLabelsPayload{
			AddOrUpdateLabels: map[string]string{
				"qpa-scale": service,
			},
		},
		ScalingConfig:      &types.NodegroupScalingConfig{
			DesiredSize: aws.Int32(size),
		},
	}

	req, err := e.Client.UpdateNodegroupConfig(context.TODO(), input)

	if err != nil {
		err = fmt.Errorf("eks %s nodegroup %s scale up fail: %s",  cluster, group, err.Error())
		return
	}

	ss, _ := json.Marshal(req)
	fmt.Println(string(ss))
	if req.Update != nil && len(req.Update.Errors) > 0 {
		message, _ := json.Marshal(req.Update.Errors)
		err = fmt.Errorf("eks %s nodegroup %s scale up fail: %s",  cluster, group, string(message))
	}
	return
}

