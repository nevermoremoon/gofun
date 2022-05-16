package aliyun

import (
	"cloud-manager/app/config"
	"cloud-manager/app/modules/console"
	"cloud-manager/app/modules/jumpserver"
	"cloud-manager/app/modules/n9e"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Ecs struct {
	Client *ecs.Client
}

func NewEcs(regionId, accessKey, secretKey string) (e *Ecs, err error) {
	ecsClient, err := ecs.NewClientWithAccessKey(
		regionId,
		accessKey,
		secretKey,
	)
	if err == nil {
		e = &Ecs{Client: ecsClient}
	}

	return e, err
}

/* API: https://help.aliyun.com/document_detail/25620.html?spm=a2c4g.11186623.6.1356.15be5fd9YJeUIr */
type InstanceType struct {
	MemorySize          string `json:"MemorySize"`
	CpuCoreCount        string `json:"CpuCoreCount"`
	InstanceTypeId      string `json:"InstanceTypeId"`
	InstanceTypeFamily  string `json:"InstanceTypeFamily"`
	//EniQuantity         string `json:"EniQuantity"`
	//GPUAmount           string `json:"GPUAmount"`
	//GPUSpec             string `json:"GPUSpec"`
	//InstancePpsRx       string `json:"InstancePpsRx"`
	//InstancePpsTx       string `json:"InstancePpsTx"`
	//InstanceBandwidthRx string `json:"InstanceBandwidthRx"`
	//InstanceBandwidthTx string `json:"InstanceBandwidthTx"`
}

type TypeData struct {
	InstanceTypes []*InstanceType `json:"InstanceTypes"`
}

func (e *Ecs) ListInstanceTypes() (interface{}, error) {
	request := ecs.CreateDescribeInstanceTypesRequest()
	response, err := e.Client.DescribeInstanceTypes(request)
	typeData := TypeData{InstanceTypes: []*InstanceType{}}
	if err == nil {
		for _, iType := range response.InstanceTypes.InstanceType {
			typeData.InstanceTypes = append(typeData.InstanceTypes, &InstanceType{
				MemorySize: fmt.Sprintf("%dG", int(iType.MemorySize)),
				CpuCoreCount: fmt.Sprintf("%dC", iType.CpuCoreCount),
				InstanceTypeId: iType.InstanceTypeId,
				InstanceTypeFamily: iType.InstanceTypeFamily,
				//EniQuantity: fmt.Sprintf("%d", iType.EniTotalQuantity),
				//GPUAmount: fmt.Sprintf("%d", iType.GPUAmount),
				//GPUSpec: iType.GPUSpec,
				//InstancePpsRx: fmt.Sprintf("%dPps", iType.InstancePpsRx),
				//InstancePpsTx: fmt.Sprintf("%dPps", iType.InstancePpsTx),
				//InstanceBandwidthRx: fmt.Sprintf("%dkbit/s", iType.InstanceBandwidthRx),
				//InstanceBandwidthTx: fmt.Sprintf("%dkbit/s", iType.InstanceBandwidthTx),
			})
		}
	}
	return typeData, err
}

/* 不使用 */
func (e *Ecs) ListInstanceTypeFamilies() (interface{}, error) {
	request := ecs.CreateDescribeInstanceTypeFamiliesRequest()
	response, err := e.Client.DescribeInstanceTypeFamilies(request)
	return response, err
}

type Image struct {
	ImageId      string `json:"ImageId"`
	OSName       string `json:"OSName"`
	Architecture string `json:"Architecture"`
	OSType       string `json:"OSType"`
	Platform     string `json:"Platform"`
	ImageName    string `json:"ImageName"`
	Size               int    `json:"Size"`
	SystemDiskSize     int    `json:"SystemDiskSize"`
	SystemDiskName     string `json:"SystemDiskName"`
	DiskDeviceMappings []*DiskDeviceMapping `json:"DiskDeviceMappings"`
	Status             string `json:"Status"`
	Progress           string `json:"Progress"`
}

type ImageData struct {
	TotalCount int        `json:"TotalCount"`
	PageNumber int        `json:"PageNumber"`
	PageSize   int        `json:"PageSize"`
	Images     []*Image   `json:"Images"`
}

type DiskDeviceMapping struct {
	Category   string     `json:"Category"`
	DeviceName string     `json:"DeviceName"`
	Type       string     `json:"Type"`
	Size       string     `json:"Size"`
	Format     string     `json:"Format"`
	SnapshotId string     `json:"SnapshotId"`
	Progress   string     `json:"Progress"`   //对于复制中的镜像，返回复制任务的进度。
	RemainTime string     `json:"RemainTime"` //对于复制中的镜像，返回复制任务的剩余时间，单位为秒。
}

func addImage(response *ecs.DescribeImagesResponse, imgList *[]*Image) {
	for _, img := range response.Images.Image {
		var diskDeviceMappings []*DiskDeviceMapping
		var systemDiskName string
		for _, disk := range img.DiskDeviceMappings.DiskDeviceMapping {
			if disk.Type == "system" {
				systemDiskName = disk.Device
			}
			diskDeviceMappings = append(diskDeviceMappings, &DiskDeviceMapping{
				DeviceName: disk.Device,
				Type:       disk.Type,
				Size:       disk.Size,
				SnapshotId: disk.SnapshotId,
				Progress:   disk.Progress,
				RemainTime: fmt.Sprintf("%d s", disk.RemainTime),
				Format: disk.Format,
			})
		}
		*imgList = append(*imgList, &Image{
			ImageId: img.ImageId,
			OSName: img.OSName,
			Architecture: img.Architecture,
			OSType: img.OSType,
			Platform: img.Platform,
			ImageName: img.ImageName,
			Size: img.Size,
			SystemDiskSize: img.Size,
			SystemDiskName: systemDiskName,
			DiskDeviceMappings: diskDeviceMappings,
			Status: img.Status,
			Progress: img.Progress,
		})
	}
}

func (e *Ecs) ListImages(imageId, pageSize, pageNumber string) (interface{}, error) {
	request := ecs.CreateDescribeImagesRequest()
	request.ImageOwnerAlias = "self"
	if imageId != "" {
		request.ImageId = imageId
	}
	request.Status="Creating,Waiting,Available,UnAvailable,CreateFailed,Deprecated"
	imageData := ImageData{Images: []*Image{}}
	if pageSize == "-1" {
		request.PageSize = requests.Integer("20")  /* 按默认值取 */
		request.PageNumber = requests.Integer("1")
	} else {
		request.PageSize = requests.Integer(pageSize)
		request.PageNumber = requests.Integer(pageNumber)
	}
	response, err := e.Client.DescribeImages(request)
	if err == nil {
		/* 加载第一次 */
		addImage(response, &imageData.Images)
		imageData.PageNumber = response.PageNumber
		imageData.PageSize = response.PageSize
		imageData.TotalCount = response.TotalCount

		/* 加载剩余所有 */
		if pageSize == "-1" {
			imageData.PageNumber = 1
			imageData.PageSize = imageData.TotalCount
			for i := 2; imageData.TotalCount > len(imageData.Images); i++ {
				request.PageNumber = requests.Integer(strconv.Itoa(i))
				response, _  = e.Client.DescribeImages(request)
				addImage(response, &imageData.Images)
			}
		}
	}
	return imageData, err
}


type CloudInstanceData struct {
	TotalCount int        `json:"TotalCount"`
	PageNumber int        `json:"PageNumber"`
	PageSize   int        `json:"PageSize"`
	Instances  []CloudInstance  `json:"Instances"`
}

type CloudInstance struct {
	Name string        `json:"Name"`
	ImageId string     `json:"ImageId"`
	PrivateIpv4 string `json:"PrivateIpv4"`
	InstanceId string  `json:"InstanceId"`
	Status string      `json:"status"`
	OperationLocks bool `json:"OperationLocks"`
}

//处理拷贝的镜像
func (e *Ecs) AutoDeleteCloneImage(imageId string) {
	var imageData ImageData
	var image *Image
	if imageId == "" {
		fmt.Printf("AutoDeleteCloneImage not provide ImageId\n")
		return
	}
	resp, err := e.ListImages(imageId, "10", "1")
	if err != nil {
		fmt.Printf("Get Image %s fail: %s\n", imageId, err.Error())
		return
	}
	imageData = resp.(ImageData)
	if len(imageData.Images) > 0 {
		image = imageData.Images[0]
	} else {
		fmt.Printf("Get Image %s empty \n", imageId)
	}

	if image != nil {
		fmt.Printf("===CopyImage %s:%s delete start...\n", image.ImageName, image.ImageId)
		_, err := e.DeleteImage(image.ImageId)
		if err != nil {
			fmt.Printf("Clone image %s:%s delete failed: %s\n", image.ImageName, image.ImageId, err.Error())
		} else {
			fmt.Printf("Clone image %s:%s delete success.\n", image.ImageName, image.ImageId)
		}
		fmt.Println("Please wait 5s...")
		time.Sleep(time.Duration(5)*time.Second)
		for _, disk := range image.DiskDeviceMappings {
			if disk.SnapshotId != "" {
				_, err := e.DeleteSnapshot(disk.SnapshotId)
				if err != nil {
					fmt.Printf("Snapshot %s delete failed: %s", disk.SnapshotId, err.Error())
				} else {
					fmt.Printf("Snapshot %s delete success.\n", disk.SnapshotId)
				}
			}
		}
		fmt.Printf("===EOF CopyImage delete end...")
	}
}

//删除快照
func (e *Ecs) DeleteSnapshot(snapshotId string ) (success bool, err error) {
	request := ecs.CreateDeleteSnapshotRequest()
	request.SnapshotId = snapshotId
	request.Force = requests.NewBoolean(true)
	_, err = e.Client.DeleteSnapshot(request)
	if err == nil {
		success = true
	}
	return
}

//删除镜像
func (e *Ecs) DeleteImage(imageId string) (success bool, err error) {
	request := ecs.CreateDeleteImageRequest()
	request.ImageId = imageId
	request.Force = requests.NewBoolean(true)
	_, err = e.Client.DeleteImage(request)
	if err == nil {
		success = true
	}
	return
}

//克隆镜像
func (e *Ecs) CreateImage(instanceId string) (imageId string, err error) {
	request := ecs.CreateCreateImageRequest()
	request.InstanceId = instanceId
	request.ImageName = fmt.Sprintf("%s-clone", instanceId)
	request.Tag = &[]ecs.CreateImageTag {
		{
			Key: "creator",
			Value: "cloud-manager",
		},
		{
			Key: "clone-from",
			Value: instanceId,
		},
	}
	response, err := e.Client.CreateImage(request)
	if err == nil {
		imageId = response.ImageId
	}
	return
}

func addInstance(response *ecs.DescribeInstancesResponse, data *CloudInstanceData) {
	for _, instance := range response.Instances.Instance {
		var lock bool
		if len(instance.OperationLocks.LockReason) > 0 {
			lock = true
			continue
		}
		if !(instance.Status == "Running" || instance.Status == "Stopped") {
			fmt.Printf("%s=%s", instance.InstanceName, instance.Status)
			continue
		}
		var ip string
		if len(instance.VpcAttributes.PrivateIpAddress.IpAddress) > 0 {
			ip = instance.VpcAttributes.PrivateIpAddress.IpAddress[0]
		}
		data.Instances = append(data.Instances, CloudInstance{
			Name: instance.InstanceName,
			ImageId: instance.ImageId,
			PrivateIpv4: ip,
			InstanceId: instance.InstanceId,
			Status: instance.Status,
			OperationLocks: lock,
		})
	}
}

func (e *Ecs) ListInstances(instanceId, pageSize, pageNumber string) (interface{}, error) {
	request := ecs.CreateDescribeInstancesRequest()
	if instanceId != "" {
		request.InstanceIds = fmt.Sprintf("[\"%s\"]", instanceId)
	}
	if pageSize == "-1" {
		request.PageSize = requests.Integer("100")  /* 按默认值取 */
		request.PageNumber = requests.Integer("1")
	} else {
		request.PageSize = requests.Integer(pageSize)
		request.PageNumber = requests.Integer(pageNumber)
	}

	cloudInstanceData := CloudInstanceData{}
	response, err := e.Client.DescribeInstances(request)
	if err == nil {
		/* 加载第一次 */
		addInstance(response, &cloudInstanceData)
		cloudInstanceData.PageNumber = response.PageNumber
		cloudInstanceData.PageSize = response.PageSize
		cloudInstanceData.TotalCount = response.TotalCount

		/* 加载剩余所有 */
		var nextToken string
		nextToken = response.NextToken
		if pageSize == "-1" {
			cloudInstanceData.PageNumber = 1
			cloudInstanceData.PageSize = cloudInstanceData.TotalCount
			for i := 2; nextToken != ""; i++ {
				request.PageNumber = requests.Integer(strconv.Itoa(i))
				response, _  = e.Client.DescribeInstances(request)
				nextToken = response.NextToken
				addInstance(response, &cloudInstanceData)
			}
		}
	}
	return cloudInstanceData, err
}

type KeyPair struct {
	KeyPairName     string  `json:"KeyPairName"`
	ResourceGroupId string  `json:"ResourceGroupId"`
}

type KeyPairData struct {
	TotalCount int         `json:"TotalCount"`
	PageNumber int         `json:"PageNumber"`
	PageSize   int         `json:"PageSize"`
	KeyPairs   []*KeyPair  `json:"KeyPairs"`
}

func addKeyPair(response *ecs.DescribeKeyPairsResponse, kpList *[]*KeyPair) {
	for _, kp := range response.KeyPairs.KeyPair {
		*kpList = append(*kpList, &KeyPair{
			KeyPairName: kp.KeyPairName,
			ResourceGroupId: kp.ResourceGroupId,
		})
	}
}


func (e *Ecs) ListKeyPairs(pageSize, pageNumber string) (interface{}, error) {
	request := ecs.CreateDescribeKeyPairsRequest()
	keypairData := KeyPairData{KeyPairs: []*KeyPair{}}
	if pageSize == "-1" {
		request.PageSize = requests.Integer("20")  /* 按默认值取 */
		request.PageNumber = requests.Integer("1")
	} else {
		request.PageSize = requests.Integer(pageSize)
		request.PageNumber = requests.Integer(pageNumber)
	}
	response, err := e.Client.DescribeKeyPairs(request)
	if err == nil {
		/* 加载第一次 */
		addKeyPair(response, &keypairData.KeyPairs)
		keypairData.PageNumber = response.PageNumber
		keypairData.PageSize = response.PageSize
		keypairData.TotalCount = response.TotalCount

		/* 加载剩余所有 */
		if pageSize == "-1" {
			keypairData.PageNumber = 1
			keypairData.PageSize = keypairData.TotalCount
			for i := 2; keypairData.TotalCount > len(keypairData.KeyPairs); i++ {
				request.PageNumber = requests.Integer(strconv.Itoa(i))
				response, _  = e.Client.DescribeKeyPairs(request)
				addKeyPair(response, &keypairData.KeyPairs)
			}
		}
	}
	return keypairData, err
}


type SecurityGroup struct {
	SecurityGroupId   string `json:"SecurityGroupId"`
	SecurityGroupName string `json:"SecurityGroupName"`
	Description       string `json:"Description"`
	SecurityGroupType string `json:"SecurityGroupType"`
	VpcId             string `json:"VpcId"`
}

type SecurityGroupData struct {
	TotalCount     int               `json:"TotalCount"`
	PageNumber     int               `json:"PageNumber"`
	PageSize       int               `json:"PageSize"`
	RegionId       string            `json:"RegionId"`
	SecurityGroups []*SecurityGroup  `json:"SecurityGroups"`
}

func addSecurityGroup(response *ecs.DescribeSecurityGroupsResponse, sgList *[]*SecurityGroup) {
	for _, sg := range response.SecurityGroups.SecurityGroup {
		*sgList = append(*sgList, &SecurityGroup{
			SecurityGroupId: sg.SecurityGroupId,
			SecurityGroupName: sg.SecurityGroupName,
			Description: sg.Description,
			SecurityGroupType: sg.SecurityGroupType,
			VpcId: sg.VpcId,
		})
	}
}

func (e *Ecs) ListSecurityGroups(vpcId, pageSize, pageNumber string) (interface{}, error) {
	request := ecs.CreateDescribeSecurityGroupsRequest()
	request.VpcId = vpcId
	sgData := SecurityGroupData{SecurityGroups: []*SecurityGroup{}}
	if pageSize == "-1" {
		request.PageSize = requests.Integer("20")  /* 按默认值取 */
		request.PageNumber = requests.Integer("1")
	} else {
		request.PageSize = requests.Integer(pageSize)
		request.PageNumber = requests.Integer(pageNumber)
	}
	response, err := e.Client.DescribeSecurityGroups(request)
	if err == nil {
		/* 加载第一次 */
		addSecurityGroup(response, &sgData.SecurityGroups)
		sgData.PageNumber = response.PageNumber
		sgData.PageSize = response.PageSize
		sgData.TotalCount = response.TotalCount
		sgData.RegionId = response.RegionId

		/* 加载剩余所有 */
		if pageSize == "-1" {
			sgData.PageNumber = 1
			sgData.PageSize = sgData.TotalCount
			for i := 2; sgData.TotalCount > len(sgData.SecurityGroups); i++ {
				request.PageNumber = requests.Integer(strconv.Itoa(i))
				response, _  = e.Client.DescribeSecurityGroups(request)
				addSecurityGroup(response, &sgData.SecurityGroups)
			}
		}
	}
	return sgData, err
}

type InstanceParam struct {
	ImageId            string     `json:"ImageId"`         /* ami */
	InstanceType       string     `json:"InstanceType"`    /* 规格 */
	Count              int        `json:"Count"`           /* 最小实例数,default=1 */
	Monitoring         bool       `json:"Monitoring"`      /* 启用CloudWatch详细监控 */
	SubnetId           string     `json:"SubnetId"`        /* 子网id */
	UserData           string     `json:"UserData"`        /* 脚本 */
	SecurityGroupId    string     `json:"SecurityGroupId"` /* 安全组列表 */
	KeyPair            string     `json:"KeyPair"`         /* 登陆密钥 */
	InstanceName       string     `json:"InstanceName"`    /* 实例名 */
	HostName           string     `json:"HostName"`        /* 主机名 */
	SystemDisk         SystemDisk `json:"SystemDisk"`
	DataDisk           []DataDisk `json:"DataDisk"`
	InstanceChargeType string     `json:"InstanceChargeType"`
	Period             string     `json:"Period"`
	PeriodUnit         string     `json:"PeriodUnit"`
	ResourceGroupId    string     `json:"ResourceGroupId"`
	Tags               []Tag      `json:"Tags"`
	DryRun             bool       `json:"DryRun"`
	SourceGroup        string     `json:"SourceGroup"`
	PublicIpv4         []string   `json:"PublicIpv4"`         //"EipAddress"
	N9eBound           []Bound    `json:"N9eBound"`           /* 挂载点 */
	FormId             int        `json:"FormId"`
	IsClone            bool       `json:"IsClone"`            //是否克隆
	CloneInstanceId    string     `json:"CloneInstanceId"`    //克隆实例的ID
}

type Bound struct {
	Nid        int       `json:"Nid"`
	Ident      string    `json:"Ident"`
	Path       string    `json:"Path"`
	BindFinish bool      `json:"BindFinish"`
	Reason     string    `json:"Reason"`
}

type JmsExecResult struct {
	Stderr string `json:"stderr"`
	Stdout string `json:"stdout"`
	Rc     int    `json:"rc"`
	Delta  string `json:"delta"`
	IsFinished bool `json:"is_finished"`
	Err    string   `json:"error"`
}

type JmsExec struct {
	Reason   string          `json:"reason"`
	Command  string          `json:"command"`
	End      bool            `json:"end"`
	Result   *JmsExecResult  `json:"result"`
	LogUrl   string          `json:"logurl"`
	HostName     string      `json:"hostname"`
	DateCreated  string      `json:"date_created"`
	DateFinished string      `json:"date_finished"`
}

type JBound struct {
	nid        int
	Path       string `json:"Path"`
	BindFinish bool   `json:"BindFinish"`
	Reason     string `json:"Reason"`
	Output     string `json:"Output"`
}

const (
	APPCENTER = "quanshi.appcenter"
	APPCENTER2 = "quanshi.app-center"
	APPBASECOMP = "quanshi.basecomp"
)

func (ipm *InstanceParam) SetHostName() {
	hostname :=  GetHostName(ipm.N9eBound, "")
	ipm.HostName = hostname
	ipm.InstanceName = hostname
}

func (b *Bound) GetHostnamePrefix() (prefix string) {
	var module, cluster string
	pathSlice := strings.Split(b.Path, ".")
	if len(pathSlice) >= 4 {
		cluster = pathSlice[len(pathSlice)-1]
		module = pathSlice[len(pathSlice)-2]
		prefix = fmt.Sprintf("%s-%s", cluster, module)
	}
	return
}


// 获取Hostname生成线索
func GetHostName(binds []Bound, privateIp string) (hostname string) {
	var bind *Bound
	var prefix string
	//既不在基础组件，也不在应用中心，则无法计算出Hostname， 成为空
	if binds != nil {
		for _, b1 := range binds {
			if strings.HasPrefix(b1.Path, APPCENTER) || strings.HasPrefix(b1.Path, APPCENTER2) {
				bind = &b1
				break
			}
		}
		if bind == nil {
			for _, b2 := range binds {
				if strings.HasPrefix(b2.Path, APPBASECOMP) {
					bind = &b2
					break
				}
			}
		}
	}
	if bind != nil {
		prefix = bind.GetHostnamePrefix()
	}

	if prefix != "" {
		hostname = prefix
		if privateIp != "" {
			ipSlice := strings.Split(privateIp, ".")
			hostname = fmt.Sprintf("%s-%s-%s", prefix, ipSlice[len(ipSlice)-2], ipSlice[len(ipSlice)-1])
		}
	}
	return
}

type Tag struct {
	Key   string `json:"key"`
	Value string `json:"Value"`
}

type SystemDisk struct {
	Size             int        `json:"size"`
	Category         string     `json:"Category"`
	PerformanceLevel string     `json:"PerformanceLevel"` /* PL0 PL1 PL2 PL3 */
	CategoryZh       string     `json:"CategoryZh"`
	DeviceName       string     `json:"DeviceName"`
}

type DataDisk struct {
	Size             int        `json:"size"`
	Category         string     `json:"Category"`
	PerformanceLevel string     `json:"PerformanceLevel"` /* PL0 PL1 PL2 PL3 */
	CategoryZh       string     `json:"CategoryZh"`
	DeviceName       string     `json:"DeviceName"`
}

var DiskZhMap map[string]string = map[string]string {
	"cloud_efficiency": "高效云盘",
	"cloud_ssd":"SSD云盘",
	"cloud_essd":"ESSD云盘",
	"cloud":"普通云盘",
}

//暂不使用
type EcsErrResponse struct {
	RequestId string  `json:"RequestId"`
	HostId    string  `json:"HostId"`
	Code      string  `json:"Code"`
	Message   string  `json:"Message"`
	Recommend string  `json:"Recommend"`
}

func getSetHostCommand(hostname string) (command string) {
	cmdfmt := "curl -Ss https://quanshi-bj.oss-cn-beijing-internal.aliyuncs.com/cloud-manager/public/set-hostname.sh | bash -s %s | tee /tmp/set-host10.txt"
	if len(config.G.AliyunInfo) > 0 && config.G.AliyunInfo[0].SetHostCommand != "" {
		cmdfmt = config.G.AliyunInfo[0].SetHostCommand
	}
	command = fmt.Sprintf(cmdfmt, hostname)
	return
}

func (ipm *InstanceParam)addSetHostnameContent() (err error) {
	if ipm.HostName == "" {
		return nil
	}
	shellPrefix := "#!/bin/bash"
	setHostCommand := getSetHostCommand(ipm.HostName)
	if strings.Contains(ipm.UserData, shellPrefix) {
		//仅替换最开头的部分
		newShellPrefix := fmt.Sprintf("%s\n%s\n", shellPrefix, setHostCommand)
		ipm.UserData = strings.Replace(ipm.UserData, shellPrefix, newShellPrefix, 1)
	} else {
		err = errors.New("[User-Data] Shell script format Illegal.")
	}
	return err
}

func (ipm *InstanceParam) checkValid() (err error) {
	//如果有公网IP，但是与申请主机数量不相等则失败
	if len(ipm.PublicIpv4) > 0 {
		if ipm.Count != len(ipm.PublicIpv4) {
			err = fmt.Errorf("Ecs count [%d] not equal public-ipv4 count [%d]", ipm.Count, len(ipm.PublicIpv4))
			return
		}
	}
	if ipm.IsClone && ipm.CloneInstanceId == "" {
		err = fmt.Errorf("Clone was selected, but the instance id to be cloned was not provided.")
		return
	}

	for _, t := range ipm.Tags {
		if t.Value == "" {
			err = fmt.Errorf("The ecs label value cannot be empty [%s=%s].", t.Key, t.Value)
			break
		}
	}
	return
}

func (e *Ecs) ListDisks(instanceId string, pageSize, pageNumber string) (interface{}, error) {
	//默认十个盘足够
	var diskDeviceMapping []DiskDeviceMapping
	if instanceId == "" {
		err := fmt.Errorf("ListDisks Param <instance-id> not provided")
		return diskDeviceMapping, err
	}

	request := ecs.CreateDescribeDisksRequest()
	request.InstanceId = instanceId
	response, err := e.Client.DescribeDisks(request)
	if err != nil {
		fmt.Printf("Fetch disk by instance-id:%s err:%s\n", instanceId, err.Error())
		return diskDeviceMapping, err
	}
	for _, disk := range response.Disks.Disk {
		diskDevice := DiskDeviceMapping{
			Category:   disk.Category,
			DeviceName: disk.Device,
			Type:       disk.Type,
			Size:       fmt.Sprintf("%d", disk.Size),
			SnapshotId: disk.SourceSnapshotId,
		}
		//插在头部
		if disk.Type == "system" {
			diskDeviceMapping = append([]DiskDeviceMapping{diskDevice}, diskDeviceMapping...)
		} else {
			diskDeviceMapping = append(diskDeviceMapping, diskDevice)
		}
	}
	return diskDeviceMapping, err
}

func (e *Ecs) imageExist(imageId string) bool {
	resp, err := e.ListImages(imageId, "10", "1")
	if err != nil {
		return false
	}
	imageData := resp.(ImageData)
	return imageData.TotalCount > 0
}

func (e *Ecs) getTryCloneImage(image, instance string) (imageId string) {
	if e.imageExist(image) {
		imageId = image
		fmt.Printf("TryRun create instance with clone is use imageid:%s from instance:%s\n", imageId, instance)
		return
	}

	resp, err := e.ListImages("", "10", "1")
	if err != nil {
		imageId = image
		return
	}
	//取得第一个镜像
	imageData := resp.(ImageData)
	if imageData.TotalCount > 0 {
		imageId = imageData.Images[0].ImageId
		imageName := imageData.Images[0].ImageName
		fmt.Printf("TryRun create instance with clone is use imageid:%s from list[0]=%s\n", imageId, imageName)
	}
	return
}

func (e *Ecs) watchCloneImage(instanceId string, insCall *InstanceCallback) (image string, err error) {
	insCall.Clone = Clone{
		CloneInstanceId: instanceId,
		Process:         "0%",
	}
	insCall.consoleClient.Callback(insCall)
	image, err = e.CreateImage(instanceId)
	if err != nil {
		return
	}
	insCall.Clone.ImageId = image
	watchTime := 120
	var state string
	var process string
	for i := 0; i < watchTime; i++ {
		resp, err := e.ListImages(image, "10", "1")
		imageData := resp.(ImageData)
		if err != nil {
			fmt.Printf("[%d] clone image <%s> get state err: **%s\n", i, image, err.Error())
		} else {
			state = imageData.Images[0].Status
			process = imageData.Images[0].Progress
			insCall.Clone.State = state
			insCall.Clone.Process = process
			insCall.consoleClient.Callback(insCall)
			fmt.Printf("[%d] clone image <%s> get state: %s, process: %s ...\n", i, image, state, process)
			if state == "Available" {
				break
			}
		}
		time.Sleep(time.Duration(30)*time.Second)
	}
	if state != "Available" {
		err = fmt.Errorf("Waiting 10mins, copy image <%s> not ready yet, give up.", image)
	}
	return
}

func (e *Ecs) CreateInstance(param interface{}) (interface{}, error) {
	defer func() {
		panicErr := recover()
		if panicErr != nil {
			fmt.Println("捕获到异常=>", panicErr)
		}
	}()
	iParam := param.(InstanceParam)

	//UserData + set-hostname
	iParam.SetHostName()
	err := iParam.addSetHostnameContent()
	if err == nil {
		err = iParam.checkValid()
	}
	b, _ := json.Marshal(iParam)
	fmt.Println("提交参数------")
	fmt.Println(string(b))
	fmt.Println("//-EOF")
	if err != nil {
		return nil, err
	}
    //克隆的回调
	instanceCallback := InstanceCallback{
		Total: iParam.Count,
		Result: fmt.Sprintf("0/%d", iParam.Count),
		Success: 0,
		IsClone: iParam.IsClone,
		callback:  !iParam.DryRun,
		Instances: []*Instance{},
		consoleClient: console.NewConosleClient(config.G.ConsoleInfo),
		FormId: iParam.FormId,
	}

	//检测被克隆的镜像是否存在
	if iParam.DryRun && iParam.IsClone {
		iParam.ImageId = e.getTryCloneImage(iParam.ImageId, iParam.CloneInstanceId)
	}
	//是否克隆
	if !iParam.DryRun && iParam.IsClone {
		imageId, err := e.watchCloneImage(iParam.CloneInstanceId, &instanceCallback)
		if err != nil {
			instanceCallback.Clone.Reason = err.Error()
			instanceCallback.SetFinish(err.Error())
			return nil, err
		}
		iParam.ImageId = imageId
	}

	var dataDisks []ecs.RunInstancesDataDisk
	for _, disk := range iParam.DataDisk {
		dataDisk := ecs.RunInstancesDataDisk{
			Size: strconv.Itoa(disk.Size),
			Category: disk.Category,
			Device: disk.DeviceName,
			//DeleteWithInstance: strconv.FormatBool(true),  默认随实例释放true
		}
		if dataDisk.Category == "cloud_essd" {
			dataDisk.PerformanceLevel = "PL0"
		}
		dataDisks = append(dataDisks, dataDisk)
	}

	var insTag []ecs.RunInstancesTag
	for _, tag := range iParam.Tags {
		insTag = append(insTag, ecs.RunInstancesTag{
			Key: tag.Key,
			Value: tag.Value,
		})
	}
	insTag = append(insTag, ecs.RunInstancesTag{
		Key: "creator",
		Value: "cloud-manager",
	})

	request := ecs.CreateRunInstancesRequest()
	request.ImageId = iParam.ImageId
	request.Amount = requests.Integer(strconv.Itoa(iParam.Count))
	if iParam.DryRun {
		//预执行， 申请数量必须是1
		request.Amount = requests.Integer("1")
	}
	//request.Amount = requests.Integer(strconv.Itoa(iParam.Count))
	request.InstanceType = iParam.InstanceType
	request.SecurityGroupId = iParam.SecurityGroupId
	request.VSwitchId = iParam.SubnetId
	request.InstanceName = iParam.InstanceName
	request.HostName = iParam.HostName
	request.SystemDiskCategory= iParam.SystemDisk.Category
	if request.SystemDiskCategory == "cloud_essd" {
		request.SystemDiskPerformanceLevel = "PL0"
	}
	request.SystemDiskSize= strconv.Itoa(iParam.SystemDisk.Size)
	//request.SystemDiskDiskName = iParam.SystemDisk.DeviceName 没有Device
	//request.SystemDiskPerformanceLevel= iParam.SystemDisk.PerformanceLevel
	request.DataDisk= &dataDisks
	request.InstanceChargeType= "PostPaid"  //按量付费
	request.DeletionProtection = requests.Boolean(strconv.FormatBool(true)) //开启删除保护
	request.UserData= base64.StdEncoding.EncodeToString([]byte(iParam.UserData))
	request.KeyPairName= iParam.KeyPair
	request.ResourceGroupId= iParam.ResourceGroupId
	request.Tag= &insTag
	request.DryRun= requests.Boolean(strconv.FormatBool(iParam.DryRun))

	response, err := e.Client.RunInstances(request)
	if err != nil {
		//instanceIds := []string{"i-2ze3m4tghvfnbr0utk25", "i-2ze3m4tghvfnbr0utk24"}
		//var instances []*Instance
		//for idx, insId := range instanceIds {
		//	instances = append(instances, NewEcsInstance(&iParam, insId, idx))
		//}
		//instance := NewEcsInstance(&iParam, instanceIds)
		//go func(instances []*Instance) {
		//	e.StartWatch(instances)
		//}(instances)
		dryFlag := "Request validation has been passed with DryRun flag set."
		if iParam.DryRun && strings.Contains(err.Error(), dryFlag) {
            /*
			instance := NewEcsInstance(&iParam, "i-2zegx6s8mwibwbdvco1z")
			go func(instance *Instance) {
				e.StartWatch(instance)
			}(instance)
            */
			fmt.Println("==DryRun create Success...")
			return "DryRun Success.", nil
		}
		instanceCallback.SetFinish(err.Error())
		return response, err
	}
	if iParam.DryRun == false {
		if len(response.InstanceIdSets.InstanceIdSet) > 0 {
			//Start Watch
			//instanceId := response.InstanceIdSets.InstanceIdSet[0]
			instanceIds := response.InstanceIdSets.InstanceIdSet
			var instances []*Instance
			for idx, insId := range instanceIds {
				instances = append(instances, NewEcsInstance(&iParam, insId, idx))
			}
			//instance := NewEcsInstance(&iParam, instanceIds)
			go func(instances []*Instance, clone *Clone) {
				e.StartWatch(instances, clone)
			}(instances, &instanceCallback.Clone)
		} else {
			err = errors.New("Ecs instance is not found, the InstanceIdSet is 0.")
		}
	}
	if err != nil {
		fmt.Println("---Create Error:", err)
	}
	return response, err
}

type InstanceData struct {
	TotalCount     int                                `json:"TotalCount"`
	PageNumber     int                                `json:"PageNumber"`
	PageSize       int                                `json:"PageSize"`
	Instances      *ecs.InstancesInDescribeInstances  `json:"Instances"`
}

type Clone struct {
	CloneInstanceId string `json:"CloneInstanceId"`
	ImageId         string `json:"ImageId"`
	State           string `json:"state"`
	Process         string `json:"Process"`
	Reason          string `json:"Reason"`
	HasDelete      bool    `json:"HasDelete"`
}

type InstanceCallback struct {
	Instances   []*Instance `json:"Instances"`
	Total       int         `json:"Total"`
	Result      string      `json:"Result"`
	Success     int         `json:"Success"`
	Reason      string      `json:"Reason"`
	End         bool        `json:"End"`
	FormId      int         `json:"FormId"`
	Clone       Clone       `json:"Clone"`
	IsClone     bool        `json:"isClone"`
	done        int
	callback    bool
	instanceIds   []string
	instanceMap   map[string]*Instance
	consoleClient *console.ConsoleClient
}


type Instance struct {
	AutoReleaseTime        string           `json:"AutoReleaseTime"`
	Cpu                    string           `json:"Cpu"`
	CreationTime           string           `json:"CreationTime"`
	DeletionProtection     bool             `json:"DeletionProtection"`
	Description            string           `json:"Description"`
	EipAddress             EipAddressAttr   `json:"EipAddress"`
	HostName               string           `json:"HostName"`
	ImageId                string           `json:"ImageId"`
	InstanceChargeType     string           `json:"InstanceChargeType"`
	InstanceId             string           `json:"InstanceId"`
	InstanceName           string           `json:"InstanceName"`
	InstanceNetworkType    string           `json:"InstanceNetworkType"`
	InstanceType           string           `json:"InstanceType"`
	InstanceTypeFamily     string           `json:"InstanceTypeFamily"`
	KeyPairName            string           `json:"KeyPairName"`
	Memory                 string           `json:"Memory"`
	OSName                 string           `json:"OSName"`
	OSType                 string           `json:"OSType"`
	PublicIpAddress        string           `json:"PublicIpAddress"`
	PrivateIpAddress       string           `json:"PrivateIpAddress"`
	RegionId               string           `json:"RegionId"`
	ZoneId                 string           `json:"ZoneId"`
	SecurityGroupIds       []string         `json:"SecurityGroupIds"`
	StartTime              string           `json:"StartTime"`
	StatusEn               string           `json:"StatusEn"`
	StatusZh               string           `json:"StatusZh"`
	Tags                   []Tag            `json:"Tags"`
	VpcAttr                VpcAttr          `json:"VpcAttr"`
	ActionStep             *ActionStep      `json:"StepAction"`
	UserData               string           `json:"UserData"`
	N9eBound               []Bound          `json:"N9eBound"`
	JmsBound               []JBound         `json:"JmsBound"`
	JmsCommand             JmsExec          `json:"JmsCommand"`
	SystemDisk             SystemDisk       `json:"SystemDisk"`
	DataDisk               []DataDisk       `json:"DataDisk"`
	FormId                 int              `json:"FormId"`
	eipFlag                bool
	IsClone                bool             `json:"IsClone"`            //是否克隆
	CloneInstanceId        string           `json:"CloneInstanceId"`    //克隆实例的ID
}


type VpcAttr struct {
	PrivateIpAddress []string `json:"PrivateIpAddress"`
	VpcId            string   `json:"VpcId"`
	VSwitchId        string   `json:"VSwitchId"`
	NatIpAddress     string   `json:"NatIpAddress"`
}

const (
	CREATE      = "Create"
	REGISTER    = "Register"
	BIND        = "Bind"
	SYNCJUMP    = "Syncjump"
	FINISH      = "Finish"
	CLONE       = "Clone"
)

var StepZhMap map[string]string = map[string]string {
    "Create": "实例创建中...",
    "Register": "n9e注册中...",
    "Bind": "n9e挂载中...",
    "Syncjump": "跳板机同步中...",
    "Finish": "完成",
    "Clone": "实例克隆中...",
}

var StepNumberMap map[string]int= map[string]int {
	"Finish":     1000,
	"Create":     1001,
	"Register":   1002,
	"Bind":       1003,
	"SyncJump":   1004,
	"Clone":      1005,
}

var InstanceStateMap map[string]string = map[string]string {
	"Pending": "创建中",
	"Running": "运行中",
	"Starting": "启动中",
	"Stopping": "停止中",
	"Stopped": "已停止",
}

type ActionStep struct {
	CurrentStep   string `json:"CurrentStep"`
	StepZh        string `json:"StepZh"`
	StepEn        string `json:"StepEn"`
	StepIndex     int    `json:"StepIndex"`
	Finish        bool   `json:"Finish"`
	Reason        string `json:"Reason"`
	//callback      bool
	//consoleClient *console.ConsoleClient
}

func NewEcsInstance(param *InstanceParam, id string, idx int) (instance *Instance) {
	instance = &Instance{
		InstanceId: id,
		N9eBound: []Bound{},
		Tags: []Tag{},
		SecurityGroupIds: []string{},
		DataDisk: []DataDisk{},
		EipAddress: EipAddressAttr{},
		ActionStep: &ActionStep{},
		IsClone: param.IsClone,
		CloneInstanceId: param.CloneInstanceId,
	}
	if param == nil {
		return
	}

	instance.FormId = param.FormId
	instance.UserData = param.UserData
	if len(param.PublicIpv4) > 0 {
		instance.EipAddress.AllocationId = param.PublicIpv4[idx]
		instance.eipFlag = true
	}
	instance.SystemDisk = SystemDisk{
		Size: param.SystemDisk.Size,
		Category: param.SystemDisk.Category,
		CategoryZh: DiskZhMap[param.SystemDisk.Category],
	}
	if param.DataDisk != nil {
		 for _, d := range param.DataDisk {
		 	instance.DataDisk = append(instance.DataDisk, DataDisk{
				Size: d.Size,
				Category: d.Category,
				CategoryZh: DiskZhMap[d.Category],
			})
		 }
	}
	if param.Tags != nil {
		for _, t := range param.Tags {
			instance.Tags = append(instance.Tags, Tag{
				Key:   t.Key,
				Value: t.Value,
			})
		}
	}
	if param.N9eBound != nil {
		for _, bind := range param.N9eBound {
			instance.N9eBound = append(instance.N9eBound, Bound {
				Nid: bind.Nid,
				Ident: bind.Ident,
				Path: bind.Path,
			})
		}
	}

	return
}

/*
func (ins *Instance) NeedEip() bool {
	if ins.EipAddress.AllocationId != "" {
		return true
	}
	return false
}
*/

func (ins *Instance) ResetHostName() {
	/*
	if ins.PrivateIpAddress == "" {
		return
	}
	ipSlice := strings.Split(ins.PrivateIpAddress, ".")
	hostname := fmt.Sprintf("%s-%s-%s", ins.HostName, ipSlice[len(ipSlice)-2], ipSlice[len(ipSlice)-1])
	ins.HostName = hostname
	ins.InstanceName = hostname
	*/
	if ins.PrivateIpAddress == "" {
		return
	}
	ipSlice := strings.Split(ins.PrivateIpAddress, ".")
	suffix := fmt.Sprintf("%s-%s", ipSlice[len(ipSlice)-2], ipSlice[len(ipSlice)-1])
	if strings.Contains(ins.HostName, suffix) {
		return
	}
	hostname := fmt.Sprintf("%s-%s", ins.HostName, suffix)
	ins.HostName = hostname
	ins.InstanceName = hostname
}

func (ins *Instance) Format(rins *ecs.Instance) {
	ins.AutoReleaseTime = rins.AutoReleaseTime
	ins.Cpu = fmt.Sprintf("%s vCpu", strconv.Itoa(rins.Cpu))
	ins.Memory = fmt.Sprintf("%s G", strconv.Itoa(rins.Memory / 1024))
	ins.OSName = rins.OSName
	ins.OSType = rins.OsType
	ins.HostName = rins.HostName
	ins.InstanceName = rins.InstanceName
	ins.ImageId = rins.ImageId
	ins.InstanceType = rins.InstanceType
	ins.KeyPairName = rins.KeyPairName
	ins.RegionId = rins.RegionId
	ins.ZoneId = rins.ZoneId
	ins.InstanceId = rins.InstanceId
	ins.InstanceTypeFamily = rins.InstanceTypeFamily
	ins.InstanceChargeType = rins.InstanceChargeType
	ins.InstanceNetworkType = rins.InstanceNetworkType
	ins.DeletionProtection = rins.DeletionProtection
	ins.CreationTime = rins.CreationTime
	ins.StartTime = rins.StartTime
	ins.Description = rins.Description
	ins.StatusEn = rins.Status

	if ins.SecurityGroupIds != nil && len(ins.SecurityGroupIds) <= 0 {
		for _, s := range rins.SecurityGroupIds.SecurityGroupId {
			ins.SecurityGroupIds = append(ins.SecurityGroupIds, s)
		}
	}
	ins.EipAddress.Bandwidth = rins.EipAddress.Bandwidth
	ins.EipAddress.IpAddress = rins.EipAddress.IpAddress
	ins.EipAddress.IsSupportUnassociate = rins.EipAddress.IsSupportUnassociate

	ins.VpcAttr  = VpcAttr{
		VpcId: rins.VpcAttributes.VpcId,
		VSwitchId: rins.VpcAttributes.VSwitchId,
		NatIpAddress: rins.VpcAttributes.NatIpAddress,
		PrivateIpAddress: rins.VpcAttributes.PrivateIpAddress.IpAddress,
	}
	/* public-ipv4, 优先查找eip，其次选择临时公网ip */
	ins.PublicIpAddress = rins.EipAddress.IpAddress
	if ins.PublicIpAddress == "" && len(rins.PublicIpAddress.IpAddress) > 0 {
		ins.PublicIpAddress = rins.PublicIpAddress.IpAddress[0]
	}
	/* internal-ipv4 优先查找vpc网络的ip */
	if len(ins.VpcAttr.PrivateIpAddress) > 0 {
		ins.PrivateIpAddress = ins.VpcAttr.PrivateIpAddress[0]
	} else {
		if len(rins.InnerIpAddress.IpAddress) > 0 {
			ins.PrivateIpAddress = rins.InnerIpAddress.IpAddress[0]
		}
	}
	if _, ok := InstanceStateMap[rins.Status]; ok {
		ins.StatusZh = InstanceStateMap[rins.Status]
	}
	//重新生成HostName
	ins.ResetHostName()

}

type EipAddressAttr struct {
	AllocationId         string `json:"AllocationId"`
	Bandwidth            int    `json:"Bandwidth "`
	InternetChargeType   string `json:"InternetChargeType"`
	IpAddress            string `json:"IpAddress"`
	IsSupportUnassociate bool   `json:"IsSupportUnassociate"`
	BindFinish           bool   `json:"BindFinish"`
	Reason               string `json:"Reason"`
}

func (e *Ecs) DescribeInstanceByIps(ips []string) (interface{}, error) {
	encJson, _ := json.Marshal(ips)
	request := ecs.CreateDescribeInstancesRequest()
	request.InstanceNetworkType = "vpc"
	request.PrivateIpAddresses = string(encJson)
	response, err := e.Client.DescribeInstances(request)
	var instanceData *InstanceData
	if err == nil {
		instanceData = &InstanceData{
			PageNumber: response.PageNumber,
			TotalCount: response.TotalCount,
			PageSize: response.PageSize,
			Instances: &response.Instances,
		}
	}
	return instanceData, err
}

//func (e *Ecs) DescribeInstance(instanceIds []string) (interface{}, error) {
func (e *Ecs) DescribeInstance(instanceIds []string) (interface{}, error) {
	encJson, _ := json.Marshal(instanceIds)
	request := ecs.CreateDescribeInstancesRequest()
	request.InstanceIds = string(encJson)
	response, err := e.Client.DescribeInstances(request)
	var instanceData *InstanceData
	if err == nil {
		instanceData = &InstanceData{
			PageNumber: response.PageNumber,
			TotalCount: response.TotalCount,
			PageSize: response.PageSize,
			Instances: &response.Instances,
		}
	}
	return instanceData, err
}

func (e *Ecs) GetInstance(instanceIds []string, userData string, n9eBound []Bound) (*InstanceData, error) {
	encjson, _ := json.Marshal(instanceIds)
	request := ecs.CreateDescribeInstancesRequest()
	request.InstanceIds = string(encjson)
	response, err := e.Client.DescribeInstances(request)
	var instances InstanceData
	if err == nil {
		fmt.Println(response.GetHttpContentString())
		instances.TotalCount = response.TotalCount
		instances.PageSize = response.PageSize
		instances.PageNumber = response.PageNumber
		//instances.ToLocal(response.Instances, userData, n9eBound)
	}
	return &instances, nil
}

func (ins *Instance)SetStep(step string) {
	if step != "" {
		ins.ActionStep.CurrentStep = step
		ins.ActionStep.StepEn = step
		ins.ActionStep.StepZh = StepZhMap[step]
		ins.ActionStep.StepIndex = StepNumberMap[step]
	}
}


func (ins *Instance)SetFinish(reason string, call InstanceCallback) {
	if reason != "" {
		ins.ActionStep.Reason = reason
	}
	ins.ActionStep.Finish = true
}

func (insCall *InstanceCallback)SetStep(step string, index int, reason string) {
	// -1 更新所有
	if index < 0 {
		for _, ins := range insCall.Instances {
			ins.SetStep(step)
		}
	} else {
		//更新单个
		insCall.Instances[index].SetStep(step)
		if step == FINISH {
			if reason != "" {
				insCall.Instances[index].ActionStep.Reason = reason
			}
			insCall.Instances[index].ActionStep.Finish = true
			//insCall.done++
		}
	}

	var done int
	var success int
	for _, ins := range insCall.Instances {
		if ins.ActionStep.Finish {
			done++
			processOk := true
			if ins.ActionStep.Reason != "" {
				processOk = false
			} else {
				if ins.EipAddress.AllocationId != "" && ins.EipAddress.Reason != "" {
					processOk = false
				}
				for _, bind := range ins.N9eBound {
					if bind.Reason != "" {
						processOk = false
						break
					}
				}
				for _, bind := range ins.JmsBound {
					if bind.Reason != "" {
						processOk = false
						break
					}
				}
			}
			if processOk {
				success += 1
			}
		}
	}
	insCall.done = done
	insCall.Success = success
	insCall.Result = fmt.Sprintf("%d/%d", insCall.Success, insCall.Total)
	if insCall.callback {
		insCall.consoleClient.Callback(*insCall)
	}
}

func (insCall *InstanceCallback)SetFinish(reason string) {
	if reason != "" {
		insCall.Reason = reason
	}
	insCall.End = true
	if insCall.callback {
		insCall.consoleClient.Callback(*insCall)
	}
}


// 废弃，hostname生效通过内部脚本来处理
func (e *Ecs) RebootInstance(instance *Instance) bool {
	//开始重启
	request := ecs.CreateRebootInstanceRequest()
	request.InstanceId = instance.InstanceId
	resp, err := e.Client.RebootInstance(request)
	_ = resp
	_ = err
	return false
}


func (e *Ecs)GetEipState(allocationId string) (bool, error) {
	var available bool
	var err       error
	request := ecs.CreateDescribeEipAddressesRequest()
	request.AllocationId = allocationId
	resp, err := e.Client.DescribeEipAddresses(request)
	fmt.Println("-Eip state---")
	b, _ := json.Marshal(resp)
	fmt.Println(string(b))
	if err == nil {
		if resp.EipAddresses.EipAddress != nil && len(resp.EipAddresses.EipAddress) > 0 {
			if resp.EipAddresses.EipAddress[0].Status ==  "Available" {
				available = true
			} else {
				instanceId := resp.EipAddresses.EipAddress[0].InstanceId
				err = errors.New(fmt.Sprintf("Eip allocationId <%s> already in-use with Ec2[%s].", allocationId, instanceId))
			}
		} else {
			err = errors.New(fmt.Sprintf("Eip allocationId <%s>  not found.", allocationId))
		}
	}
	fmt.Printf("Eip是否可用: %t\n", available)
	return available, err
}

func (e *Ecs) AssociateEipAddress(instance *Instance) error {
	fmt.Println("---AssociateEipAddress start...")
	//保证未被使用
	_, err := e.GetEipState(instance.EipAddress.AllocationId)
	if err != nil {
		instance.EipAddress.Reason = err.Error()
		return err
	}
	request := ecs.CreateAssociateEipAddressRequest()
	request.InstanceId = instance.InstanceId
	request.AllocationId = instance.EipAddress.AllocationId
	resp, err := e.Client.AssociateEipAddress(request)
	if err != nil {
		instance.EipAddress.Reason = err.Error()
	} else {
		instance.EipAddress.BindFinish = true
	}
	fmt.Println("---AssociateEipAddress finish...")
	_ = resp
	return err
}


func (e *Ecs) DescribeInstanceType(insType string) (out string) {
	request := ecs.CreateDescribeInstanceTypesRequest()
	request.InstanceTypes = &([]string{insType})
	resp, err := e.Client.DescribeInstanceTypes(request)
	if err == nil {
		for _, iType := range resp.InstanceTypes.InstanceType {
			if iType.InstanceTypeId == insType {
				out = fmt.Sprintf("%dvCPU_%dGiB", iType.CpuCoreCount, int(iType.MemorySize))
				break
			}
		}
	}
	return
}
func (e *Ecs) GetInstanceByIp(ip string) (attr map[string]string, err error) {
	var instance *ecs.Instance
	var responseData interface{}
	responseData, err = e.DescribeInstanceByIps([]string{ip})
	if err != nil {
		return
	}
	instanceData := responseData.(*InstanceData)
	if len(instanceData.Instances.Instance) <= 0 {
		err = fmt.Errorf("Aliyun instance [ip:%s] not found.", ip)
		return
	} else {
		instance = &instanceData.Instances.Instance[0]
	}
	ourInstance := Instance{}
	ourInstance.Format(instance)
	attr = map[string]string {
		"vendor": "ali",
		"region": ourInstance.RegionId,
		"id": ourInstance.InstanceId,
		"ip": ourInstance.PrivateIpAddress,
		"name": instance.InstanceName,
		"public_ipv4": ourInstance.PublicIpAddress,
		"family": ourInstance.InstanceType,
	}
	attr["resource"]=e.DescribeInstanceType(ourInstance.InstanceType)
	return
}

func (e *Ecs) UpdateInstanceByIp(ip, hostname string) (err error) {
	var instance *ecs.Instance
	var responseData interface{}
	responseData, err = e.DescribeInstanceByIps([]string{ip})
	if err != nil {
		return
	}
	instanceData := responseData.(*InstanceData)
	if len(instanceData.Instances.Instance) <= 0 {
		err = fmt.Errorf("Aliyun instance [ip:%s] not found.", ip)
		return
	} else {
		instance = &instanceData.Instances.Instance[0]
	}
	request := ecs.CreateModifyInstanceAttributeRequest()
	request.InstanceId = instance.InstanceId
	request.InstanceName = hostname
	response, err := e.Client.ModifyInstanceAttribute(request)
	if err != nil {
		fmt.Printf("Modify instance hostname [%s] failed: %s\n", hostname, err.Error())
		if response != nil {
			fmt.Println(response.GetHttpContentString())
		}
	}
	return
}

//TODO：更新更多内容cloud-manager
func (e *Ecs) UpdateInstance(instance *Instance) bool {
	fmt.Printf("---Start update instance attr <%s>\n", instance.HostName)
	request := ecs.CreateModifyInstanceAttributeRequest()
	request.InstanceId = instance.InstanceId
	request.HostName = instance.HostName
	request.InstanceName = instance.InstanceName
	response, err := e.Client.ModifyInstanceAttribute(request)
	if err != nil {
		fmt.Printf("Modify instance hostname [%s] failed: %s\n", instance.HostName, err.Error())
		if response != nil {
			fmt.Println(response.GetHttpContentString())
		}
		return false
	}
	fmt.Println("---Finish update instance Attr")
	return true
}

func (e *Ecs) StartWatch(instances []*Instance, clone *Clone) {
	fmt.Println("Starting watch instance after 5s...")
	time.Sleep(time.Duration(5)*time.Second)
	var imageId string
	instanceCallback := InstanceCallback{
		Instances: instances,
		instanceMap: map[string]*Instance{},
		Total: len(instances),
		Clone: *clone,
		callback:  true,
		consoleClient: console.NewConosleClient(config.G.ConsoleInfo),
	}
	if clone.CloneInstanceId != "" {
		imageId = clone.ImageId
		instanceCallback.IsClone = true
	}

	for _, ins := range instances {
		instanceCallback.instanceIds = append(instanceCallback.instanceIds, ins.InstanceId)
		instanceCallback.instanceMap[ins.InstanceId] = ins
		if instanceCallback.FormId <= 0 {
			instanceCallback.FormId = ins.FormId
		}
	}
	watchTime := 60

	var message string
	instanceCallback.SetStep(CREATE, -1, "")
	// Wait 5 min
	for i:= 0; i <= watchTime; i++ {
		var readyCount int
		responseData, err := e.DescribeInstance(instanceCallback.instanceIds)
		instanceData := responseData.(*InstanceData)
		if err != nil {
			message = fmt.Sprintf("Get Instance %v state err: %s\n", instanceCallback.instanceIds, err.Error())
			instanceCallback.SetFinish(message)
			fmt.Println(message)
			return
		}
		if instanceData.TotalCount <= 0 {
			message = fmt.Sprintf("[%d] Get instance %v state empty: try again, waitting 3s ...\n", i, instanceCallback.instanceIds)
			fmt.Println(message)
			time.Sleep(time.Duration(3)*time.Second)
			continue
		}
		//列表对应关系,更新机器状态
		for _, ins := range instanceData.Instances.Instance {
			if _, ok := instanceCallback.instanceMap[ins.InstanceId]; ok {
				instanceCallback.instanceMap[ins.InstanceId].Format(&ins)
				if instanceCallback.instanceMap[ins.InstanceId].StatusEn == "Running" {
					readyCount += 1
				}
			}
		}

        //公网IP刚刚绑定，需要多循环一次
		var again bool
		for _, ins := range instanceCallback.Instances {
			if ins.eipFlag && ins.StatusEn == "Running" {
				_ = e.AssociateEipAddress(ins)
				//获取最后一次状态
				ins.eipFlag = false
				again = true
			}
		}

		/* 机器就绪 */
		fmt.Printf("[%d] Goroutine: wait instance running %d/%d ...\n", i+1, readyCount, instanceCallback.Total)
		if again {
			fmt.Println("公网IP绑定，重新获取一次状态...")
			continue
		}
		if readyCount == instanceCallback.Total {
			break
		}
		time.Sleep(time.Duration(5)*time.Second)
	}
	fmt.Println("Stop watch instance, waiting init.sh exec 180s...")
	//统一等待180s，等待脚本执行完成
	time.Sleep(time.Duration(180)*time.Second)
	for i := 0; i < len(instanceCallback.Instances); i++ {
		ins := instanceCallback.Instances[i]
		if ins.StatusEn == "" {
			message = fmt.Sprintf("Get instance state err: not found instance <%s> state.\n", ins.InstanceId)
			instanceCallback.SetStep(FINISH, i, message)
			fmt.Println(message)
		}
		e.UpdateInstance(ins)
		fmt.Println()
		if ins.StatusEn == "Running" {
			wait := len(instanceCallback.Instances) - i
			if wait < 1 {
				wait = 1
			}
			fmt.Printf(">>>>>>>Wait userdate exec finish...%ds <<<<<<<", wait * 5)
			time.Sleep(time.Duration(wait*5)*time.Second)
			e.RegisterN9e(&instanceCallback, i)
			e.RegisterJms(&instanceCallback, i)
			instanceCallback.SetStep(FINISH, i, "")
		}
	}
	if instanceCallback.IsClone {
		e.AutoDeleteCloneImage(imageId)
		instanceCallback.Clone.HasDelete = true
	}
	instanceCallback.SetFinish("")
}

func (e *Ecs) RegisterJms(insCallback *InstanceCallback, index int) {
	instance := insCallback.Instances[index]
	insCallback.SetStep(SYNCJUMP, index, "")
	fmt.Printf("Host[%s]<%s> Start Jms Register...\n", instance.HostName,instance.PrivateIpAddress)
	n9eCli := n9e.NewN9EClient(config.G.N9eInfo, config.DEFAULT)
	for i:= 0; i < len(instance.JmsBound); i++ {
		//TODO: jump server
		_, err := n9eCli.RegisterJms(instance.PrivateIpAddress,instance.HostName, "ali", []int{instance.JmsBound[i].nid})
		if err != nil {
			instance.JmsBound[i].Reason = err.Error()
		}
		instance.JmsBound[i].BindFinish = true
	}

	//开始尝试获取脚本输出
	var err  error
	var host *jumpserver.AssetsHost
	jmsCli := jumpserver.NewJmsClient(config.G.JumpInfo, config.DEFAULT)
	fmt.Printf(">>> Get [%s] shell exec output...\n", instance.PrivateIpAddress)
	instance.JmsCommand= JmsExec{}
	host, err = jmsCli.NewAssetsHost(instance.PrivateIpAddress,instance.HostName)
	if err == nil && host != nil {
		var connectivity bool
		for i:= 0; i < 5; i++ {
			fmt.Printf("[%d] Host [%s] test connect send, wait 20s...\n", i+1, host.Ip)
			connectivity, err = host.TestConnect(jmsCli)
			if err != nil {
				instance.JmsCommand.Reason = err.Error()
				break
			}
			if connectivity {
				break
			}
		}
		//经过前面的等待，不可连接，也不阻止，继续执行,这样可获取不可连接的准确原因
		if !connectivity {
			instance.JmsCommand.Reason = fmt.Sprintf("机器不可连接")
		}
		var resp *jumpserver.JmsAnsibleResponse
		resp, err = jmsCli.ExecCommand([]string{host.Id}, "cat /tmp/ali-init10.log")
		if err == nil {
			//等待20s
			taskId := resp.Id
			waitTime := 10
			for j := 0; j < waitTime; j++ {
				output, err := jmsCli.GetExecCommand(taskId)
				fmt.Printf("[%d] watch jms ansible exec... 3s ...\n", j)
				if err != nil {
					//有一次错误，则停止
					instance.JmsCommand.Reason = err.Error()
					break
				} else {
					instance.JmsCommand.LogUrl = fmt.Sprintf("%s%s", config.G.JumpInfo.EndPoint, output.LogUrl)
					instance.JmsCommand.Command = output.Command
					instance.JmsCommand.DateCreated = output.DateCreated
					instance.JmsCommand.DateFinished = output.DateFinished
					//只有1台主机
					for k, v := range output.Result{
						if v.Err != "" {
							//去除前端符号
							re, _ := regexp.Compile("\\<|\\>")
							v.Err =  re.ReplaceAllString(v.Err, "")
						}
						instance.JmsCommand.Result = &JmsExecResult{
							Stderr: v.Stderr,
							Stdout: v.Stdout,
							Rc:     v.Rc,
							Delta:  v.Delta,
							IsFinished: output.IsFinished,
							Err: v.Err,
						}
						instance.JmsCommand.HostName = k
						break
					}
					if output.IsFinished {
						break
					}
				}
				time.Sleep(time.Duration(3)*time.Second)
			}
		}
	}
	if err != nil {
		instance.JmsCommand.Reason = err.Error()
	} else {
		if host == nil {
			instance.JmsCommand.Reason = fmt.Sprintf("Exec command, host [%s] not found.\n", instance.PrivateIpAddress)
		}
	}
	instance.JmsCommand.End = true
	fmt.Printf("<<< Get [%s] shell exec output done...\n", instance.PrivateIpAddress)
}

func (e *Ecs) RegisterN9e(insCallback *InstanceCallback, index int) {
	instance := insCallback.Instances[index]
	fmt.Printf("Host[%s]<%s> Start n9e Register...\n", instance.HostName,instance.PrivateIpAddress)

	hostForm := n9e.HostRegisterForm{
		IP: instance.PrivateIpAddress,
		Ident: instance.PrivateIpAddress,
		Name: instance.HostName,
	}
	var message string

	if hostForm.Name == "" {
		hostForm.Name = hostForm.IP
	}
	n9eCli := n9e.NewN9EClient(config.G.N9eInfo, config.DEFAULT)
	//instance.SetStep(REGISTER)
	insCallback.SetStep(REGISTER, index, "")
	host, err := n9eCli.RegisterHost(hostForm)
    if err != nil {
		message = fmt.Sprintf("Host [%s]<%s> register fail: %s\n", instance.PrivateIpAddress, instance.InstanceId, err.Error())
		//instance.SetFinish(message)
		insCallback.SetStep(FINISH, index, message)
		fmt.Println(message)
		return
	}

	fmt.Printf("Host [%s]<%s> register success...\n", host.IP, instance.InstanceId)
	//TODO: 租户填入配置文件
	resp := host.SetHostTenant(config.MAJOR, true, n9eCli)
	if resp.Success {
		// 完成所有
		fmt.Printf("Host [%s]<%s> set tenant success...\n",  host.IP, instance.InstanceId)
		//var nidList []int
		insCallback.SetStep(BIND, index, message)
		for i:= 0; i < len(instance.N9eBound); i++ {
			resp := host.HostBind(instance.N9eBound[i].Nid, n9eCli)
			instance.N9eBound[i].BindFinish = resp.Success
			instance.N9eBound[i].Reason = resp.Err
			//成功一个，将来jms注册一个
			if resp.Success {
				//nid := instance.N9eBound[i].Nid
				//nidList = append(nidList, nid)
				//不再处理机器中心
				if strings.HasPrefix(instance.N9eBound[i].Path, "quanshi.machinecenter") {
					continue
				}
				instance.JmsBound = append(instance.JmsBound, JBound{
					nid:        instance.N9eBound[i].Nid,
					Path:       "",
					BindFinish: false,
					Reason:     "",
					Output:     "",
				})
			}
		}

		//instance.SetStep(SYNCJUMP)
		//TODO: jump server
		//if nidList != nil && len(nidList) > 0 {
		//	_ = n9eCli.RegisterJms(instance.PrivateIpAddress,instance.HostName, "ali", nidList )
		//}
		//insCallback.SetStep(FINISH, index, "")
	} else {
		message = fmt.Sprintf("Host [%s]<%s> set tenant fail: %s\n",  host.IP, instance.InstanceId, resp.Err)
		insCallback.SetStep(FINISH, index, message)
		fmt.Println(message)
	}
}

type TerminateResponse struct {
	Total         int      `json:"Total"`
	Finish        bool     `json:"Finish"`
	Reason        string   `json:"Reason"`
	FormId        int      `json:"FormId"`
	InstanceIds   []string `json:"InstanceIds"`
	InstanceIps   []string `json:"InstanceIps"`
	TerminateInstances  []*TerminateInstanceResult `json:"TerminateInstances"`
	callback      bool
	consoleClient *console.ConsoleClient
}

type TerminateInstanceResult struct {
	InstanceId    string `json:"InstanceId"`
	PrivateIpv4   string `json:"PrivateIpv4"`
	CurrentStateEn  string `json:"CurrentStateEn"`
	CurrentStateZh  string `json:"CurrentStateZh"`
	N9eOffLine      *N9eOffLine `json:"N9eOffLine"`
}

type N9eOffLine struct {
	Ip      string   `json:"ip"`
	Ident   string   `json:"Ident"`
	AmsId   int      `json:"AmsId"`
	RdbId   int      `json:"RdbId"`
	OffLine bool     `json:"OffLine"`
	Reason  string   `json:"Reason"`
}

type TerminateRequest struct{
	InstanceIps []string  `json:"InstanceIps"`
	AccessKey   string    `json:"AccessKey"`
	SecretKey   string    `json:"SecretKey"`
	FormId      int       `json:"FormId"`
	DryRun      bool      `json:"DryRun"`
}

func (tr *TerminateResponse)Informer() {
	if tr.callback {
		tr.consoleClient.Callback(tr)
	}
}

func (tr *TerminateResponse)SetFinish(reason string) {
	if reason != "" {
		tr.Reason = reason
	}
	tr.Finish = true
	tr.Informer()
}

func (e *Ecs)OfflineFromN9e(tResponse *TerminateResponse) {
	n9eCli := n9e.NewN9EClient(config.G.N9eInfo, config.DEFAULT)
	for i:= 0; i < len(tResponse.TerminateInstances); i++ {
		ip := tResponse.TerminateInstances[i].PrivateIpv4
		offline := N9eOffLine {
			Ip: ip,
		}
		host := n9eCli.NewHost(ip)
		if host != nil {
			offline.Ident = host.Ident
			offline.AmsId = host.AmsID
			offline.RdbId = host.RdbID
			err := host.Offline(n9eCli)
			if err != nil {
				offline.Reason = err.Error()
			} else {
				offline.OffLine = true
				_ = n9eCli.OfflineJms(host.IP, host.Hostname)
			}
		} else {
			offline.Reason = fmt.Sprintf("The host <%s> not found.", ip)
		}
		tResponse.TerminateInstances[i].N9eOffLine = &offline
	}
}

func (e *Ecs)WatchInstanceState(tResponse *TerminateResponse) {
	fmt.Println("Start watch terminate ecs state...")
	tResponse.Informer()
	var instanceState map[string]string
	idsJson, _ := json.Marshal(tResponse.InstanceIds)
	input := ecs.CreateDescribeInstancesRequest()
	input.PageSize = requests.Integer("20")
	input.InstanceIds = string(idsJson)

	watchTime := 60
	var hasOk int
	for i := 0; i <= watchTime; i++ {
		fmt.Printf("[%d] Goroutine filter watch...\n", i+1)
		instanceState = map[string]string{}
		resp, err := e.Client.DescribeInstances(input)

		if err != nil {
			tResponse.SetFinish(err.Error())
			return
		} else {
			for _, ins := range resp.Instances.Instance {
				instanceState[ins.InstanceId] = ins.Status
			}
		}
		// 请空，分析并更新状态
		hasOk = 0
		for i:= 0; i < len(tResponse.TerminateInstances); i++ {
			if tResponse.TerminateInstances[i].CurrentStateEn == "Stopped" {
				hasOk++
				continue
			}
			instanceId := tResponse.TerminateInstances[i].InstanceId
			state, ok := instanceState[instanceId]
			if ok {
				tResponse.TerminateInstances[i].CurrentStateEn = state
				tResponse.TerminateInstances[i].CurrentStateZh = InstanceStateMap[state]
				if state == "Stopped" {
					hasOk++
				}
			} else {
				//不存在，认为已经释放删除
				tResponse.TerminateInstances[i].CurrentStateEn = "Stopped"
				tResponse.TerminateInstances[i].CurrentStateZh = "已停止"
				hasOk++
			}
		}
		fmt.Println("Has OK: ", hasOk)
		if hasOk == len(tResponse.TerminateInstances) {
			e.OfflineFromN9e(tResponse)
			tResponse.SetFinish("")
			return
		}
		tResponse.Informer()
		time.Sleep(time.Duration(5)*time.Second)
	}
	// 监听结束
	if !tResponse.Finish {
		e.OfflineFromN9e(tResponse)
		tResponse.SetFinish("3min watch stop, please go to the console to view")
	}
}

func (e *Ecs)AuthToTerminateInstances(param interface{}, region string) (interface{}, error) {
	//TODO: 重新认证
	tRequest := param.(TerminateRequest)
	var err error

	terminateResponse := TerminateResponse{
		consoleClient: console.NewConosleClient(config.G.ConsoleInfo),
		callback: !tRequest.DryRun,
		InstanceIps: tRequest.InstanceIps,
		Total: len(tRequest.InstanceIps),
		FormId: tRequest.FormId,
	}

	if tRequest.SecretKey == "" || tRequest.AccessKey == "" {
		err = errors.New("The credentials info is illegal, please check.")
	} else {
		if len(tRequest.InstanceIps) <= 0 {
			err = errors.New("The InstanceIps is empty set [], please check.")
		} else if len(tRequest.InstanceIps) > 20 {
			err = errors.New("The InstanceIps can support less than 20 one time.")
		}
	}
	if err != nil {
		terminateResponse.SetFinish(err.Error())
		return nil, err
	}
	ipsJson, _ := json.Marshal(tRequest.InstanceIps)
	input := ecs.CreateDescribeInstancesRequest()
	input.PageSize = requests.Integer("20")
	input.PrivateIpAddresses = string(ipsJson)
	response, err := e.Client.DescribeInstances(input)
	if err != nil {
		terminateResponse.SetFinish(err.Error())
		return nil, err
	}

	var instanceIds []string
	instanceIdMap := map[string]string{}
	for _, ins := range response.Instances.Instance {
		id := ins.InstanceId
		//只取vpc网络下的内网IP
		var ip, state string
		if len(ins.VpcAttributes.PrivateIpAddress.IpAddress) > 0 {
			ip = ins.VpcAttributes.PrivateIpAddress.IpAddress[0]
		}
		state = ins.Status
		instanceIds = append(instanceIds, ins.InstanceId)
		instanceIdMap[ins.InstanceId] = ip
		terminateResponse.TerminateInstances = append(terminateResponse.TerminateInstances, &TerminateInstanceResult{
			InstanceId:     id,
			PrivateIpv4:    ip,
			CurrentStateEn: state,
			CurrentStateZh: InstanceStateMap[state],
		})
	}
	terminateResponse.InstanceIds = instanceIds

	fmt.Println(instanceIdMap)
	if len(instanceIds) != len(tRequest.InstanceIps) {
		err = errors.New("Filter instance by ip, result count not equal ip.")
		terminateResponse.SetFinish(err.Error())
		return nil, err
	}
	//重新认证客户端
	var client *ecs.Client
    //TODO
	client, err = ecs.NewClientWithAccessKey(
		region,
		tRequest.AccessKey,
		tRequest.SecretKey,
	)
	if err != nil {
		terminateResponse.SetFinish(err.Error())
		return nil, err
	}

	request := ecs.CreateDeleteInstancesRequest()
	request.InstanceId = &instanceIds
	request.Force = requests.Boolean("true") //强制释放运行中（Running）的实例
	request.DryRun = requests.Boolean(fmt.Sprintf("%t", tRequest.DryRun))
	fmt.Println("------TResponse---")
    b, _ := json.Marshal(terminateResponse)
    fmt.Println(string(b))
	fmt.Println("------End---")
	resp, err := client.DeleteInstances(request)
	//预执行，在成功的情况下也是有err的，特征明显，所以无错误一定不是预执行
	if err != nil {
		dryFlag := "This request is a dryrun request with successful result."
		if tRequest.DryRun && strings.Contains(err.Error(), dryFlag) {
			fmt.Println("==DryRun terminate Success...")
			return "DryRun Success.", nil
		}
		terminateResponse.SetFinish(err.Error())
	} else {
		go func(tResponse *TerminateResponse) {
			e.WatchInstanceState(tResponse)
		}(&terminateResponse)
	}
	if err != nil {
		fmt.Println("---Terminate Error:", err)
	}
	return resp, err
}