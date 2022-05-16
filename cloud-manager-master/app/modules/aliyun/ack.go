package aliyun

import (
	"encoding/json"
	"fmt"
	cs "github.com/alibabacloud-go/cs-20151215/v2/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/client"
	"github.com/alibabacloud-go/tea/tea"
)

type ACK struct {
	Client *cs.Client
}

type NodeGroup struct {
	Cluster string    `json:"cluster"`
	Name string       `json:"name"`
	GroupID   string  `json:"groupId"`
	ClusterID string  `json:"clusterId"`
}


func NewACK(regionId, accessKey, secretKey string) (a *ACK, err error) {
	endpoint := fmt.Sprintf("cs.%s.aliyuncs.com", regionId)
	config := &openapi.Config{
		AccessKeyId: tea.String(accessKey),
		AccessKeySecret: tea.String(secretKey),
		Endpoint:  tea.String(endpoint),
	}
	// 访问的域名
	csClient, err := cs.NewClient(config)

	if err == nil {
		a = &ACK{Client: csClient}
	}
	return
}

func (a *ACK) getNodeGroup(clusterName, groupName string) (nodegroup *NodeGroup, err error) {
	var clusterId string
	request := &cs.DescribeClustersRequest{}
	request.SetName(clusterName)
	resp1, err := a.Client.DescribeClusters(request)
	if err != nil {
		err = fmt.Errorf("ack %s get cluster id err: %s", clusterName, err.Error())
		return
	}

	if resp1 != nil {
		for _, cluster := range resp1.Body {
			if tea.StringValue(cluster.Name) == clusterName {
				clusterId = tea.StringValue(cluster.ClusterId)
				break
			}
		}
	}

	if clusterId == "" {
		err = fmt.Errorf("ack %s not found cluster id", clusterName)
		return
	}

	resp2, err := a.Client.DescribeClusterNodePools(tea.String(clusterId))
	if err != nil {
		err = fmt.Errorf("ack %s get nodegroup %s err: %s", clusterName, groupName, err.Error())
		return
	}

	if resp2 != nil && resp2.Body != nil {
		for _, nodePool := range resp2.Body.Nodepools {
			if nodePool.NodepoolInfo != nil && tea.StringValue(nodePool.NodepoolInfo.Name) == groupName {
				nodegroup = &NodeGroup {
					Cluster: clusterName,
					Name:    groupName,
					GroupID: tea.StringValue(nodePool.NodepoolInfo.NodepoolId),
					ClusterID: clusterId,
				}
				return
			}
		}
	}
	err = fmt.Errorf("ack %s not found nodegroup %s", clusterName, groupName)
	return
}

func (a *ACK) NodeGroupScale(cluster, group, service string, number int32) (err error){
	nodegroup , err := a.getNodeGroup(cluster, group)
	if err != nil {
		return
	}
	s, _ := json.Marshal(nodegroup)
	fmt.Println(string(s))

	size := int64(number)
	request := &cs.ScaleClusterNodePoolRequest{}
	request.SetCount(size)

	resp, err := a.Client.ScaleClusterNodePool(tea.String(nodegroup.ClusterID), tea.String(nodegroup.GroupID), request)
	if err != nil {
		err = fmt.Errorf("ack %s nodegroup %s scale up fail: %s",  cluster, group, err.Error())
		return
	}

	if resp != nil {
		output, _ := json.Marshal(resp)
		fmt.Println(string(output))
	}
	return
}
