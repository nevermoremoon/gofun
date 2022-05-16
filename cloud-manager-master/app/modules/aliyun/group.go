package aliyun

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/services/resourcemanager"
)

type Group struct {
	Client *resourcemanager.Client
}

func NewGroup(regionId, accessKey, secretKey string) (g *Group, err error) {
	groupClient, err := resourcemanager.NewClientWithAccessKey(
		regionId,
		accessKey,
		secretKey,
	)
	if err == nil {
		g = &Group{Client: groupClient}
	}
	return g, err
}

type ResourceGroup struct {
	Name         string `json:"Name"`
	Id           string `json:"Id"`
	DisplayName  string `json:"DisplayName"`
}

type GroupData struct {
	TotalCount       int               `json:"TotalCount"`
	PageNumber       int               `json:"PageNumber"`
	PageSize         int               `json:"PageSize"`
	ResourceGroups   []*ResourceGroup  `json:"ResourceGroups"`
}

func (g *Group) ListGroups() (interface{}, error) {
	request := resourcemanager.CreateListResourceGroupsRequest()
	// 必须启用https
	request.Scheme="https"
	groupData := GroupData{ResourceGroups: []*ResourceGroup{}}
	response, err := g.Client.ListResourceGroups(request)
	if err == nil {
		groupData.PageNumber = response.PageNumber
		groupData.PageSize = response.PageSize
		groupData.TotalCount = response.TotalCount
		for _, rg := range response.ResourceGroups.ResourceGroup {
			groupData.ResourceGroups = append(groupData.ResourceGroups, &ResourceGroup{
				Name: rg.Name,
				Id: rg.Id,
				DisplayName: rg.DisplayName,
			})
		}
	}
	return groupData, err
}
