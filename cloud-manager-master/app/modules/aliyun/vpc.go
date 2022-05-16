package aliyun

import (
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"strconv"
)


type Vpc struct {
	Client *vpc.Client
}


func NewVpc(regionId, accessKey, secretKey string) (v *Vpc, err error) {
	vpcClient, err := vpc.NewClientWithAccessKey(
		regionId,
		accessKey,
		secretKey,
	)
	if err == nil {
		v = &Vpc{Client: vpcClient}
	}

	return v, err
}

type VpcData struct {
	TotalCount int        `json:"TotalCount"`
	PageNumber int        `json:"PageNumber"`
	PageSize   int        `json:"PageSize"`
	Vpcs       []*VpcInfo `json:"Vpcs"`
}
type VpcInfo struct {
	VpcId     string  `json:"VpcId"`
	RegionId  string  `json:"RegionId"`
	VpcName   string  `json:"VpcName"`
	CidrBlock string  `json:"CidrBlock"`
}

func (v *Vpc) ListVpcs(pageSize, pageNumber string) (interface{}, error) {
	request := vpc.CreateDescribeVpcsRequest()
	//vpc数量少，不分页返回所有.

	request.PageSize = requests.Integer("10")
	request.PageNumber = requests.Integer("1")
	response, err := v.Client.DescribeVpcs(request)

	vpcData := VpcData{Vpcs: []*VpcInfo{}}
	if err == nil {
		for _, v := range response.Vpcs.Vpc {
			vpcData.Vpcs = append(vpcData.Vpcs, &VpcInfo{
				VpcId: v.VpcId,
				RegionId: v.RegionId,
				VpcName: v.VpcName,
				CidrBlock: v.CidrBlock,
			})
		}
		vpcData.PageNumber = response.PageNumber
		vpcData.PageSize = response.PageSize
		vpcData.TotalCount = response.TotalCount
	}
	return vpcData, err
}

type VSwitch struct {
	VpcId        string  `json:"VpcId"`
	VSwitchName  string  `json:"VSwitchName"`
	VSwitchId    string  `json:"VSwitchId"`
	CidrBlock    string  `json:"CidrBlock"`
	ZoneId       string  `json:"ZoneId"`
	Status       string  `json:"Status"`
}


type VswData struct {
	TotalCount int        `json:"TotalCount"`
	PageNumber int        `json:"PageNumber"`
	PageSize   int        `json:"PageSize"`
	VSwitches  []*VSwitch `json:"Subnets"`
}

func addVSwitch(response *vpc.DescribeVSwitchesResponse, vswList *[]*VSwitch) {
	for _, vsw := range response.VSwitches.VSwitch {
		*vswList = append(*vswList, &VSwitch{
			VpcId:       vsw.VpcId,
			VSwitchId:   vsw.VSwitchId,
			VSwitchName: vsw.VSwitchName,
			CidrBlock:   vsw.CidrBlock,
			ZoneId:      vsw.ZoneId,
			Status:      vsw.Status,
		})
	}
}

// 范例，暂且不用
func (v *Vpc) FilterSubnetId() (subnetId []string) {
	request := vpc.CreateListTagResourcesRequest()
	request.ResourceType = "VSWITCH"
	request.Tag = &[]vpc.ListTagResourcesTag{
		{
			Key: "cloud-manager",
			Value: "true",
		},
	}
	response, err := v.Client.ListTagResources(request)
	if err == nil {
		for _, subnet := range response.TagResources.TagResource {
			subnetId = append(subnetId, subnet.ResourceId)
		}
	} else {
		fmt.Printf("Aliyun filter subnet Id by tag:<cloud-manager=true> Error: %s", err.Error())
	}
	return subnetId
}

func (v *Vpc) ListSubnets(vpcId, pageSize, pageNumber string) (interface{}, error) {
	request := vpc.CreateDescribeVSwitchesRequest()
	vswData := VswData{VSwitches: []*VSwitch{}}
	//subnetId := v.FilterSubnetId()
	request.VpcId = vpcId
	request.Tag = &[]vpc.DescribeVSwitchesTag{
		{
			Key: "cloud-manager",
			Value: "true",
		},
	}

	if pageSize == "-1" {
		request.PageSize = requests.Integer("20")  /* 按默认值取 */
		request.PageNumber = requests.Integer("1")
	} else {
		request.PageSize = requests.Integer(pageSize)
		request.PageNumber = requests.Integer(pageNumber)
	}
	response, err := v.Client.DescribeVSwitches(request)
    if err == nil {
    	/* 加载第一次 */
        addVSwitch(response, &vswData.VSwitches)
		vswData.PageNumber = response.PageNumber
		vswData.PageSize = response.PageSize
		vswData.TotalCount = response.TotalCount
		currentTotal := len(response.VSwitches.VSwitch)
		/* 加载剩余所有 */
		if pageSize == "-1" {
			vswData.PageNumber = 1
			vswData.PageSize = vswData.TotalCount
			for i := 2; vswData.TotalCount > currentTotal; i++ {
				request.PageNumber = requests.Integer(strconv.Itoa(i))
				response, _  = v.Client.DescribeVSwitches(request)
				currentTotal += len(response.VSwitches.VSwitch)
				addVSwitch(response, &vswData.VSwitches)
			}
		}
		//真实数量
		vswData.TotalCount = len(vswData.VSwitches)
	}

	return vswData, err
}

type  EipAddress struct {
	AllocationId  string `json:"Id"`
	Name          string `json:"Name"`
	IpAddress     string `json:"IpAddress"`
	Bandwidth     string `json:"Bandwidth"`
    InstanceId    string `json:"InstanceId"`
	InUse         bool   `json:"InUse"`
}

type EipData struct {
	TotalCount   int           `json:"TotalCount"`
	PageNumber   int           `json:"PageNumber"`
	PageSize     int           `json:"PageSize"`
	EipAddresses []*EipAddress `json:"EipAddresses"`
}
/* Status
Associating：绑定中。
Unassociating：解绑中。
InUse：已分配。
Available：可用
*/

func addEipAddress(response *vpc.DescribeEipAddressesResponse, eipList *[]*EipAddress) {
	for _, eip := range response.EipAddresses.EipAddress {
		var inUse bool
		if eip.Status != "Available" {
			inUse = true
		}
		*eipList = append(*eipList, &EipAddress{
			AllocationId: eip.AllocationId,
			Name: eip.Name,
			IpAddress: eip.IpAddress,
			Bandwidth: fmt.Sprintf("%sM", eip.Bandwidth),
			InstanceId: eip.InstanceId,
			InUse: inUse,
		})
	}
}


func (v *Vpc) ListPublicIpv4s(pageSize, pageNumber string) (interface{}, error) {
	request := vpc.CreateDescribeEipAddressesRequest()
	eipData := EipData{EipAddresses: []*EipAddress{}}
	if pageSize == "-1" {
		request.PageSize = requests.Integer("20")  /* 按默认值取 */
		request.PageNumber = requests.Integer("1")
	} else {
		request.PageSize = requests.Integer(pageSize)
		request.PageNumber = requests.Integer(pageNumber)
	}
	response, err := v.Client.DescribeEipAddresses(request)
	if err == nil {
		/* 加载第一次 */
		addEipAddress(response, &eipData.EipAddresses)
		eipData.PageNumber = response.PageNumber
		eipData.PageSize = response.PageSize
		eipData.TotalCount = response.TotalCount

		/* 加载剩余所有 */
		if pageSize == "-1" {
			eipData.PageNumber = 1
			eipData.PageSize = eipData.TotalCount
			for i := 2; eipData.TotalCount > len(eipData.EipAddresses); i++ {
				request.PageNumber = requests.Integer(strconv.Itoa(i))
				response, _  = v.Client.DescribeEipAddresses(request)
				addEipAddress(response, &eipData.EipAddresses)
			}
		}
	}
	//排除掉已使用的
	eipAvailableData := EipData{
		PageNumber: eipData.PageNumber,
		PageSize: eipData.PageSize,
		EipAddresses: []*EipAddress{},
	}
	for _, eip := range eipData.EipAddresses {
		if eip.InUse {
			continue
		}
		eipAvailableData.EipAddresses = append(eipAvailableData.EipAddresses, eip)
	}
	eipAvailableData.TotalCount = len(eipAvailableData.EipAddresses)


	return eipAvailableData.EipAddresses, err
}
