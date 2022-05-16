package aws

import (
	"cloud-manager/app/config"
	"cloud-manager/app/modules/console"
	"cloud-manager/app/modules/jumpserver"
	"cloud-manager/app/modules/n9e"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Ec2 struct {
	Client *ec2.Client
}


func NewEc2(awsClient *ec2.Client) (*Ec2, error) {
	return &Ec2{Client: awsClient}, nil
}


func (e *Ec2) ListInstanceTypeFamilies() (interface{}, error) {
	return nil, errors.New("Not Support this interface.")
}

type KeyPair struct {
	KeyName    *string `json:"KeyName"`
	KeyPairId  *string `json:"KeyPairId"`
}

type KeyPairData struct {
	TotalCount  int        `json:"TotalCount"`
	KeyPairs    []*KeyPair `json:"KeyPairs"`
}

func (e *Ec2) ListKeyPairs(pageSize, pageNumber string) (interface{}, error) {
	input := &ec2.DescribeKeyPairsInput{}
	req, err := e.Client.DescribeKeyPairs(context.TODO(), input)
	keypairData := KeyPairData{KeyPairs: []*KeyPair{}}
	if err == nil {
		keypairData.TotalCount = len(req.KeyPairs)
		for _, kp := range req.KeyPairs {
			keypairData.KeyPairs = append(keypairData.KeyPairs, &KeyPair{
				KeyName: kp.KeyName,
				KeyPairId: kp.KeyPairId,
			})
		}
	}
	return keypairData, err
}

type InstanceType struct {
	InstanceType    string `json:"InstanceType"`
	MemorySize      string `json:"MemorySize"`
	CpuVCpus        string `json:"CpuVCpus"`
}

type TypeData struct {
	TotalCount    int             `json:"TotalCount"`
	InstanceTypes []*InstanceType `json:"InstanceTypes"`
}

func (e *Ec2) ListInstanceTypes() (interface{}, error) {
	input := &ec2.DescribeInstanceTypesInput{}
	req, err := e.Client.DescribeInstanceTypes(context.TODO(), input)
	typeData := TypeData{InstanceTypes: []*InstanceType{}}
	if err == nil {
		typeData.TotalCount = len(req.InstanceTypes)
		for _, it := range req.InstanceTypes {
			typeData.InstanceTypes = append(typeData.InstanceTypes, &InstanceType{
				InstanceType: string(it.InstanceType),
				MemorySize: fmt.Sprintf("%dG", int(*(it.MemoryInfo.SizeInMiB))/1024),
				CpuVCpus: fmt.Sprintf("%dC", int(*(it.VCpuInfo.DefaultVCpus))),
			})
		}
	}
	return typeData, err
}


type Image struct {
	ImageId             string `json:"ImageId"`
	Architecture        string  `json:"Architecture"`
	Platform            string `json:"OSType"`
	ImageName           string `json:"ImageName"`
	Description         string `json:"Description"`
	RootDeviceName      string `json:"RootDeviceName"`
	RootDeviceType      string  `json:"RootDeviceType"`
	State               string  `json:"State"`
	Name                string  `json:"Name"`
	BlockDeviceMapping  []*DeviceMapping `json:"BlockDeviceList"`
}

type DeviceMapping struct {
	DeviceName string `json:"DeviceName"`
	DeviceType string  `json:"DeviceType"`
	Ebs        *Ebs    `json:"Ebs"`
}

type Ebs struct {
	DeleteOnTermination bool    `json:"DeleteOnTermination"`
	Encrypted           bool    `json:"Encrypted"`
	VolumeSize          int32   `json:"VolumeSize"`
	VolumeType          string  `json:"VolumeType"`
	SnapshotId          string  `json:"SnapshotId"`
	VolumeId            string  `json:"VolumeId"`
}

type ImageData struct {
	TotalCount int        `json:"TotalCount"`
	Images     []*Image   `json:"Images"`
}
/* 数据量过大,不过滤公共镜像 */
func (e *Ec2) ListImages(imageId, pageSize, pageNumber string) (interface{}, error) {
	input := &ec2.DescribeImagesInput{
		Filters: []types.Filter{
			{
				Name: aws.String("owner-id"),
				Values: []string{
					"389125581212",   //owner
					//"141808717104", //Linux2
					//"918309763551", //amazon
				},
			},
		},
	}
	if imageId != "" {
		input.Filters = append(input.Filters, types.Filter{
		    Name: aws.String("image-id"),
			Values: []string{
				imageId,
			},
		})
	}
	req, err := e.Client.DescribeImages(context.TODO(), input)
	imageData := ImageData{Images: []*Image{}}
	if err == nil {
		imageData.TotalCount = len(req.Images)
		for _, img := range req.Images {
			var deviceList []*DeviceMapping
			for _, bd := range img.BlockDeviceMappings {
				device := &DeviceMapping{
					DeviceName: aws.ToString(bd.DeviceName),
					DeviceType: "data",
					Ebs: &Ebs{
						Encrypted: bd.Ebs.Encrypted,
						DeleteOnTermination: bd.Ebs.DeleteOnTermination,
						VolumeSize: bd.Ebs.VolumeSize,
						VolumeType: string(bd.Ebs.VolumeType),
						SnapshotId: aws.ToString(bd.Ebs.SnapshotId),
					},
				}
				/* 根磁盘始终排第一位 */
				if aws.ToString(bd.DeviceName) == aws.ToString(img.RootDeviceName) {
					device.DeviceType = "system"
					deviceList = append([]*DeviceMapping{device}, deviceList...)
				} else {
					deviceList = append(deviceList, device)
				}
			}
			//这里调换位置，ImageName存放label-name
			var labelName string
			for _, tag := range img.Tags {
				if aws.ToString(tag.Key) == "Name" {
					labelName = aws.ToString(tag.Value)
					break
				}
			}
			imageData.Images = append(imageData.Images, &Image{
				ImageId: aws.ToString(img.ImageId),
				Architecture: string(img.Architecture),
				ImageName: labelName,
				Platform:  aws.ToString(img.PlatformDetails),
				Description:  aws.ToString(img.Description),
				RootDeviceName:  aws.ToString(img.RootDeviceName),
				RootDeviceType: string(img.RootDeviceType),
				BlockDeviceMapping: deviceList,
				Name: aws.ToString(img.Name),
				State: string(img.State),
			})
		}
	}
	return imageData, err

}

type Snapshot struct {
	Progress            string
	SnapshotId          string
	State               string
	StateMessage        string
	VolumeId            string
	VolumeSize          int32
}
//list SnapshotId
func (e *Ec2) DescribeSnapshot(snapshotIds []string) ([]Snapshot, error){
	var snapshots []Snapshot
	input := &ec2.DescribeSnapshotsInput{
		SnapshotIds: snapshotIds,
	}
	resp, err := e.Client.DescribeSnapshots(context.TODO(), input)
	if err == nil {
		for _, sn := range resp.Snapshots{
			snapshots = append(snapshots, Snapshot{
				Progress:     aws.ToString(sn.Progress),
				SnapshotId:   aws.ToString(sn.SnapshotId),
				State:        string(sn.State),
				StateMessage: aws.ToString(sn.StateMessage),
				VolumeId:     aws.ToString(sn.VolumeId),
				VolumeSize:   sn.VolumeSize,
			})
		}
	}
	return snapshots, err
}

//删除快照
func (e *Ec2) DeleteSnapshot(snapshotId string) (success bool, err error) {
	input :=&ec2.DeleteSnapshotInput{
		SnapshotId: aws.String(snapshotId),
	}
	_, err = e.Client.DeleteSnapshot(context.TODO(), input)
	if err == nil {
		success = true
	}
	return
}
//删除镜像
func (e *Ec2) DeleteImage(imageId string) (success bool, err error) {

	input := &ec2.DeregisterImageInput{
		ImageId: aws.String(imageId),
	}
	_, err = e.Client.DeregisterImage(context.TODO(), input)
	if err == nil {
		success = true
	}
	return
}
//自动移除
//处理拷贝的镜像
func (e *Ec2) AutoDeleteCloneImage(imageId string) {
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
		for _, disk := range image.BlockDeviceMapping {
			if disk.Ebs.SnapshotId != "" {
				_, err := e.DeleteSnapshot(disk.Ebs.SnapshotId)
				if err != nil {
					fmt.Printf("Snapshot %s delete failed: %s", disk.Ebs.SnapshotId, err.Error())
				} else {
					fmt.Printf("Snapshot %s delete success.\n", disk.Ebs.SnapshotId)
				}
			}
		}
		fmt.Printf("===EOF CopyImage delete end...")
	}
}

//克隆镜像
func (e *Ec2) CreateImage(instanceId string) (imageId string, err error) {
	imageName := fmt.Sprintf("%s-clone", instanceId)
	input := &ec2.CreateImageInput{
		Description: aws.String(fmt.Sprintf("Created by CreateImage with instance(%s) ", instanceId)),
		InstanceId: aws.String(instanceId),
		Name: aws.String(imageName),
		NoReboot: true,
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceType("image"),
				Tags: []types.Tag{
					{
						Key:   aws.String("creator"),
						Value: aws.String("cloud-manager"),
					},
					{
						Key: aws.String("clone-from"),
						Value: aws.String(instanceId),
					},
					{
						Key: aws.String("Name"),
						Value: aws.String(imageName),
					},
				},
			},
		},
	}
	response, err := e.Client.CreateImage(context.TODO(), input)
	if err == nil {
		imageId = aws.ToString(response.ImageId)
	}
	return
}


func (e *Ec2) imageExist(imageId string) bool {
	resp, err := e.ListImages(imageId, "10", "1")
	if err != nil {
		return false
	}
	imageData := resp.(ImageData)
	return imageData.TotalCount > 0
}

func (e *Ec2) getTryCloneImage(image, instance string) (imageId string) {
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

func (e *Ec2) watchCloneImage(instanceId string, insCall *InstanceCallback) (image string, err error) {
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
	for i := 0; i < watchTime; i++ {
		resp, err := e.ListImages(image, "10", "1")
		imageData := resp.(ImageData)
		if err != nil {
			fmt.Printf("[%d] clone image <%s> get state err: **%s\n", i, image, err.Error())
		} else {
			state = imageData.Images[0].State
			var snapshotIds []string
			var process string
			var err2 error
			for _, disk := range imageData.Images[0].BlockDeviceMapping {
				snapshotIds = append(snapshotIds, disk.Ebs.SnapshotId)
			}
			if len(snapshotIds) > 0 {
				insCall.Clone.Snapshots, err2 = e.DescribeSnapshot(snapshotIds)
				if err2 != nil {
					fmt.Println(snapshotIds, "describe snapshot err: ", err)
				} else {
					for _, sn := range insCall.Clone.Snapshots {
						if process == "" {
							process = sn.Progress
						} else {
							processInt, _ := strconv.Atoi(strings.TrimRight(sn.Progress, "%"))
							lastProcessInt , _ := strconv.Atoi(process)
							//取进度最慢的快照
							if processInt < lastProcessInt {
								process = sn.Progress
							}
						}
					}
				}
			}
			if process == "" {
				process = fmt.Sprintf("%d%%", i)
			}
			if state == "available" {
				process = "100%"
			}
			insCall.Clone.State = state
			insCall.Clone.Process = process
			insCall.consoleClient.Callback(insCall)
			fmt.Printf("[%d] clone image <%s> get state: %s, process: %s ...\n", i, image, state, process)
			if state == "available" {
				break
			}
		}
		time.Sleep(time.Duration(30)*time.Second)
	}
	if state != "available" {
		err = fmt.Errorf("Waiting 10mins, copy image <%s> not ready yet, give up.", image)
	}
	return
}

type SecurityGroup struct {
	SecurityGroupId   string `json:"SecurityGroupId"`
	SecurityGroupName string `json:"SecurityGroupName"`
	Description       string `json:"Description"`
	VpcId             string `json:"VpcId"`
}

type SecurityGroupData struct {
	TotalCount     int               `json:"TotalCount"`
	SecurityGroups []*SecurityGroup  `json:"SecurityGroups"`
}

func (e *Ec2) ListSecurityGroups(vpcId, pageSize, pageNumber string) (interface{}, error) {
	/* 过滤值参考命令行输出， 全部小写，多单词加"-" */
	input := &ec2.DescribeSecurityGroupsInput{
		Filters: []types.Filter{
			{
				Name: aws.String("vpc-id"),
				Values: []string{
					vpcId,
				},
			},
		},
	}
	sgData := SecurityGroupData{SecurityGroups: []*SecurityGroup{}}
	req, err := e.Client.DescribeSecurityGroups(context.TODO(), input)
	if err == nil {
		sgData.TotalCount = len(req.SecurityGroups)
		for _, sg := range req.SecurityGroups {
			sgData.SecurityGroups = append(sgData.SecurityGroups, &SecurityGroup{
				SecurityGroupId: aws.ToString(sg.GroupId),
				SecurityGroupName: aws.ToString(sg.GroupName),
				Description: aws.ToString(sg.Description),
				VpcId: aws.ToString(sg.VpcId),
			})
		}
	}
	return sgData, err
}

const (
	APPCENTER = "quanshi.appcenter"
	APPCENTER2 = "quanshi.app-center"
	APPBASECOMP = "quanshi.basecomp"
	DRYFLAG = "Request would have succeeded, but DryRun flag is set."
)

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

/* 参数说明 https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_RunInstances.html */
type InstanceParam struct {
	ImageId         string     `json:"ImageId"`         /* ami */
	InstanceType    string     `json:"InstanceType"`    /* 规格 */
	Count           int        `json:"Count"`           /* 最大实例数,default=1 */
	Monitoring      bool       `json:"Monitoring"`      /* 启用CloudWatch详细监控 */
	SubnetId        string     `json:"SubnetId"`        /* 子网id */
	UserData        string     `json:"UserData"`        /* 脚本 */
	SecurityGroupId []string   `json:"SecurityGroupId"` /* 安全组列表 */
	KeyPair         string     `json:"KeyPair"`         /* 登陆密钥 */
	Disk            []DataDisk `json:"Disk"`            /* 磁盘数据 */
	Tags            []Tag      `json:"Tags"`            /* 标签数据 */
	DryRun          bool       `json:"DryRun"`          /* 预执行 */
	N9eBound        []Bound    `json:"N9eBound"`        /* 挂载点 */
	PublicIpv4      []string   `json:"PublicIpv4"`
	FormId          int        `json:"FormId"`
	HostName        string     `json:"HostName"`
	InstanceName    string     `json:"InstanceName"`
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
	Stderr string   `json:"stderr"`
	Stdout string   `json:"stdout"`
	Rc     int      `json:"rc"`
	Delta  string   `json:"delta"`
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
	Path       string   `json:"Path"`
	BindFinish bool     `json:"BindFinish"`
	Reason     string   `json:"Reason"`
}

type Tag struct {
	Key   string `json:"key"`
	Value string `json:"Value"`
}

type DataDisk struct {
	DeviceName       string     `json:"DeviceName"`
	VolumeSize       int32      `json:"VolumeSize"`
	VolumeType       string     `json:"VolumeType"`
}

func (ipm *InstanceParam) SetHostName() {
	hostname :=  GetHostName(ipm.N9eBound, "")
	ipm.HostName = hostname
	ipm.InstanceName = hostname
	//Aws 机制，增加标签
	if hostname != "" {
		ipm.Tags = append(ipm.Tags, Tag{
			Key: "Name",
			Value: hostname,
		})
	}
}

func getSetHostCommand(hostname string) (command string) {
	cmdfmt := "curl -Ss https://s3.cn-north-1.amazonaws.com.cn/hub.quanshi.com/cloud-manager/public/set-hostname.sh | bash -s %s | tee /tmp/set-host10.txt"
	if len(config.G.AwsInfo) > 0 && config.G.AwsInfo[0].SetHostCommand != "" {
		cmdfmt = config.G.AwsInfo[0].SetHostCommand
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
			err = fmt.Errorf("Ec2 count [%d] not equal public-ipv4 count [%d]", ipm.Count, len(ipm.PublicIpv4))
			return
		}
	}
	if ipm.IsClone && ipm.CloneInstanceId == "" {
		err = fmt.Errorf("Clone was selected, but the instance id to be cloned was not provided.")
		return
	}

	for _, t := range ipm.Tags {
		if t.Value == "" {
			err = fmt.Errorf("The ec2 label value cannot be empty [%s=%s].", t.Key, t.Value)
			break
		}
	}
	return
}

func (e *Ec2) CreateInstance(param interface{}) (interface{}, error) {
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
	fmt.Println("提交参数------")
	b, _ := json.Marshal(iParam)
	fmt.Println(string(b))
	fmt.Println("//--EOF")
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
	//开始克隆
	if !iParam.DryRun && iParam.IsClone {
		imageId, err := e.watchCloneImage(iParam.CloneInstanceId, &instanceCallback)
		if err != nil {
			instanceCallback.Clone.Reason = err.Error()
			instanceCallback.SetFinish(err.Error())
			return nil, err
		}
		iParam.ImageId = imageId
	}

	var dataDisks []types.BlockDeviceMapping
	for _, disk := range iParam.Disk {
		dataDisks = append(dataDisks, types.BlockDeviceMapping{
			DeviceName: aws.String(disk.DeviceName),
			Ebs: &types.EbsBlockDevice{
				VolumeType: types.VolumeType(disk.VolumeType),
				VolumeSize: disk.VolumeSize,
				DeleteOnTermination: true, //实例终止时，自动删除
			},
		})
	}
	var insTag []types.TagSpecification
	var Tags []types.Tag
	for _, tag := range iParam.Tags {
		Tags = append(Tags, types.Tag{
			Key: aws.String(tag.Key),
			Value: aws.String(tag.Value),
		})
	}

	Tags = append(Tags, types.Tag{
		Key: aws.String("creator"),
		Value: aws.String("cloud-manager"),
	})
	insTag = []types.TagSpecification{
		{
			ResourceType: types.ResourceType("instance"),
			Tags: Tags,
		},
	}

	input := &ec2.RunInstancesInput{
		ImageId: aws.String(iParam.ImageId),
		InstanceType: types.InstanceType(iParam.InstanceType),
		MinCount: int32(iParam.Count), //最小实例数，在最大实例数超出可用区库存后，选择MinCount<N<MaxCount的数量启动，如果库存比该数值还小，则不启动任何实例
		DisableApiTermination: true,   //开启终止保护
		MaxCount: int32(iParam.Count), //需要的实例数， 让他等于MinCount，则可保证库存不足，不创建任何实例
		Monitoring:  &(types.RunInstancesMonitoringEnabled{Enabled: iParam.Monitoring}),
		SubnetId: aws.String(iParam.SubnetId),
		UserData: aws.String(base64.StdEncoding.EncodeToString([]byte(iParam.UserData))),
		SecurityGroupIds: iParam.SecurityGroupId,
		KeyName: aws.String(iParam.KeyPair),
		BlockDeviceMappings: dataDisks,
		TagSpecifications: insTag,
		DryRun: iParam.DryRun,
	}

	resp, err := e.Client.RunInstances(context.TODO(), input)

	if err != nil {
		//dryFlag := "Request would have succeeded, but DryRun flag is set."
		if input.DryRun && strings.Contains(err.Error(), DRYFLAG) {
			//instanceIds := []string{"i-0040223595b1732f6", "i-0c661e3724fab9f64"}
			//var instances []*Instance
			//for idx, insId := range instanceIds {
			//	instances = append(instances, NewEc2Instance(&iParam, insId, idx))
			//}
			//go func(instances []*Instance) {
			//	e.StartWatch(instances)
			//}(instances)
			fmt.Println("==DryRun create Success...")
			return "DryRun Success.", nil
		}
		return resp, err
	}
	if iParam.DryRun == false {
		if len(resp.Instances) > 0 {
			//Start Watch
			//instanceIds := aws.ToString(resp.Instances[0].InstanceId)
			instanceIds := resp.Instances
			var instances []*Instance
			for idx, awsInstance := range instanceIds {
				instances = append(instances, NewEc2Instance(&iParam, aws.ToString(awsInstance.InstanceId), idx))
			}
			go func(instances []*Instance, clone *Clone) {
				e.StartWatch(instances, clone)
			}(instances, &instanceCallback.Clone)
		} else {
			err = errors.New("Ec2 instance is not found, the InstanceIdSet is 0.")
		}
	}
	if err != nil {
		fmt.Println("---Create Error:", err)
	}
	return resp, err
}

type InstanceData struct {
	Instances  []types.Instance  `json:"Instances"`
}
type NetworkInterface struct {
	PrivateIpAddress   string   `json:"PrivateIpAddress"`
	NetworkInterfaceId string   `json:"NetworkInterfaceId"`
	VpcId              string   `json:"VpcId"`
	SubnetId           string   `json:"SubnetId"`
	MacAddress         string   `json:"MacAddress"`
	Status             string   `json:"Status"`
	PrivateDnsName     string   `json:"PrivateDnsName"`
}

type EipAddressAttr struct {
	AllocationId         string `json:"AllocationId"`
	IpAddress            string `json:"IpAddress"`
	DnsName              string `json:"PublicDnsName"`
	BindFinish           bool   `json:"BindFinish"`
	Reason               string `json:"Reason"`
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

const (
	CREATE      = "Create"
	REGISTER    = "Register"
	BIND        = "Bind"
	SYNCJUMP    = "Syncjump"
	FINISH      = "Finish"
)

var StepZhMap map[string]string = map[string]string {
	"Create": "实例创建中...",
	"Register": "n9e注册中...",
	"Bind": "n9e挂载中...",
	"Syncjump": "跳板机同步中...",
	"Finish": "完成",
}

var StepNumberMap map[string]int= map[string]int {
	"Finish":     1000,
	"Create":     1001,
	"Register":   1002,
	"Bind":       1003,
	"SyncJump":   1004,
}

func (e *Ec2) ListDisks(instanceId string, pageSize, pageNumber string) (interface{}, error) {
	var diskDeviceMapping []DeviceMapping
	if instanceId == "" {
		err := fmt.Errorf("ListDisks Param <instance-id> not provided")
		return diskDeviceMapping, err
	}

	input1 := &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceId},
	}
	resp1, err1 := e.Client.DescribeInstances(context.TODO(), input1)
	var systemDeviceName string

	deviceMap := make(map[string]DeviceMapping)
	if err1 != nil {
		return diskDeviceMapping, err1
	} else {
		for _, r := range resp1.Reservations {
			for _, ins := range r.Instances {
				systemDeviceName = aws.ToString(ins.RootDeviceName)
				for _, device := range ins.BlockDeviceMappings {
					if device.Ebs.VolumeId != nil {
						deviceType := "data"
						if aws.ToString(device.DeviceName) == systemDeviceName {
							deviceType = "system"
						}
						deviceMap[aws.ToString(device.Ebs.VolumeId)] = DeviceMapping{
							DeviceName: aws.ToString(device.DeviceName),
							DeviceType: deviceType,
							Ebs: &Ebs{
								DeleteOnTermination: device.Ebs.DeleteOnTermination,
							},
						}
					}
				}
				break
			}
		}
	}

	input2 := &ec2.DescribeVolumesInput{
		Filters: []types.Filter{
			{
				Name: aws.String("attachment.instance-id"),
				Values: []string{instanceId},
			},
		},
	}
	resp2, err := e.Client.DescribeVolumes(context.TODO(), input2)
	if err != nil {
		fmt.Printf("Fetch disk by instance-id:%s err:%s\n", instanceId, err.Error())
		return diskDeviceMapping, err
	}
	for _, disk := range resp2.Volumes {
		deviceMapping , ok := deviceMap[aws.ToString(disk.VolumeId)]
		if ok {
			deviceMapping.Ebs.VolumeId = aws.ToString(disk.VolumeId)
			deviceMapping.Ebs.VolumeType = string(disk.VolumeType)
			deviceMapping.Ebs.SnapshotId = aws.ToString(disk.SnapshotId)
			deviceMapping.Ebs.VolumeSize = disk.Size
			if deviceMapping.DeviceName == systemDeviceName {
				diskDeviceMapping = append([]DeviceMapping{deviceMapping}, diskDeviceMapping...)
			} else {
				diskDeviceMapping = append(diskDeviceMapping, deviceMapping)
			}
		}
	}
	return diskDeviceMapping, err
}




func (ins *Instance)SetStep(step string) {
	if step != "" {
		ins.ActionStep.CurrentStep = step
		ins.ActionStep.StepEn = step
		ins.ActionStep.StepZh = StepZhMap[step]
		ins.ActionStep.StepIndex = StepNumberMap[step]
	}
}

func (ins *Instance)SetFinish(reason string) {
	if reason != "" {
		ins.ActionStep.Reason = reason
	}
	ins.ActionStep.Finish = true
}

var InstanceStateMap map[string]string = map[string]string {
	"pending": "启动中",
	"running": "运行中",
	"shutting-down": "释放前关机中",
	"terminated": "已释放",
	"stopping": "准备停止",
	"stopped": "已停止",
}

type Clone struct {
	CloneInstanceId string `json:"CloneInstanceId"`
	ImageId         string `json:"ImageId"`
	State           string `json:"state"`
	Process         string `json:"Process"`
	Reason          string `json:"Reason"`
	HasDelete       bool   `json:"HasDelete"`
	Snapshots       []Snapshot `json:"Snapshots"`
}

type InstanceCallback struct {
	Instances     []*Instance `json:"Instances"`
	Total         int         `json:"Total"`
	Result        string      `json:"Result"`
	Success       int         `json:"Success"`
	Reason        string      `json:"Reason"`
	End           bool        `json:"End"`
	FormId        int         `json:"FormId"`
	Clone         Clone       `json:"Clone"`
	IsClone       bool        `json:"isClone"`
	done          int
	callback      bool
	instanceIds   []string
	instanceMap   map[string]*Instance
	consoleClient *console.ConsoleClient
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

type Instance struct {
	Cpu                    string              `json:"Cpu"`
	LaunchTime             string              `json:"LaunchTime"`
	DeletionProtection     bool                `json:"DeletionProtection"`
	Description            string              `json:"Description"`
	EipAddress             EipAddressAttr      `json:"EipAddress"`
	HostName               string              `json:"HostName"`
	ImageId                string              `json:"ImageId"`
	InstanceId             string              `json:"InstanceId"`
	InstanceName           string              `json:"InstanceName"`
	InstanceType           string              `json:"InstanceType"`
	InstanceTypeFamily     string              `json:"InstanceTypeFamily"`
	KeyPairName            string              `json:"KeyPairName"`
	Memory                 string              `json:"Memory"`
	OSName                 string              `json:"OSName"`
	KernelId               string              `json:"KernelId"`
	OSType                 string              `json:"OSType"`
	PublicIpAddress        string              `json:"PublicIpAddress"`
	PublicDnsName          string              `json:"PublicDnsName"`
	PrivateIpAddress       string              `json:"PrivateIpAddress"`
	PrivateDnsName         string              `json:"PrivateDnsName"`
	RegionId               string              `json:"RegionId"`
	ZoneId                 string              `json:"ZoneId"`
	StatusEn               string              `json:"StatusEn"`
	StatusZh               string              `json:"StatusZh"`
	StatusReason           string              `json:"StatusReason"`
	Tags                   []Tag               `json:"Tags"`
	NetworkInterfaces      []NetworkInterface  `json:"NetworkInterfaces"`
	ActionStep             *ActionStep         `json:"StepAction"`
	UserData               string              `json:"UserData"`
	N9eBound               []Bound             `json:"N9eBound"`
	JmsBound               []JBound            `json:"JmsBound"`
	JmsCommand             JmsExec             `json:"JmsCommand"`
	SystemDisk             DataDisk            `json:"SystemDisk"`
	DataDisk               []DataDisk          `json:"DataDisk"`
	Monitoring             string              `json:"Monitoring"`
	SecurityGroupList      []SecurityGroup     `json:"SecurityGroupList"`
	FormId                 int                 `json:"FormId"`
	eipFlag                bool
}

func NewEc2Instance(param *InstanceParam, id string, idx int) (instance *Instance) {
	instance = &Instance{
		InstanceId: id,
		N9eBound: []Bound{},
		Tags: []Tag{},
		NetworkInterfaces: []NetworkInterface{},
		DataDisk: []DataDisk{},
		EipAddress: EipAddressAttr{},
		SecurityGroupList: []SecurityGroup{},
		ActionStep: &ActionStep{},
	}
	if param == nil {
		return
	}

	instance.FormId = param.FormId
	instance.UserData = param.UserData
	instance.KeyPairName = param.KeyPair
	//TODO
	//instance.RegionId = config.G.AwsInfo.RegionId
	// Eip ID
	if len(param.PublicIpv4) > 0 {
		instance.EipAddress.AllocationId = param.PublicIpv4[idx]
		instance.eipFlag = true
	}
	if len(param.Disk) > 0 {
		instance.SystemDisk = DataDisk{
			VolumeSize: param.Disk[0].VolumeSize,
			VolumeType: param.Disk[0].VolumeType,
			DeviceName: param.Disk[0].DeviceName,
		}
	}
	if len(param.Disk) > 1 {
		for i:= 1; i < len(param.Disk); i++ {
			instance.DataDisk = append(instance.DataDisk, DataDisk{
				VolumeSize: param.Disk[i].VolumeSize,
				VolumeType: param.Disk[i].VolumeType,
				DeviceName: param.Disk[i].DeviceName,
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
	if len(param.SecurityGroupId) > 0 {
		for _, sg := range param.SecurityGroupId {
			instance.SecurityGroupList = append(instance.SecurityGroupList, SecurityGroup{
				SecurityGroupId: sg,
			})
		}
	}
	return
}


func CreateContainerNode() {
	//Update
}

func (ins *Instance) ResetHostName() {
	hostName := GetHostName(ins.N9eBound, ins.PrivateIpAddress)
	if ins.PrivateIpAddress != "" {
		ins.HostName = hostName
		ins.InstanceName = hostName
	}
}

func (ins *Instance) Format(rins *types.Instance) {
	ins.LaunchTime = aws.ToTime(rins.LaunchTime).String()
	ins.Cpu = fmt.Sprintf("%d vCpu", int(rins.CpuOptions.CoreCount * rins.CpuOptions.ThreadsPerCore))
	ins.Memory = fmt.Sprintf("%s G", "")
	ins.OSName = string(rins.Architecture)
	ins.OSType = string(rins.Hypervisor)
	ins.HostName = aws.ToString(rins.PrivateDnsName)
	ins.ImageId = aws.ToString(rins.ImageId)
	ins.InstanceType = string(rins.InstanceType)
	ins.InstanceTypeFamily = string(rins.InstanceLifecycle)
	ins.KeyPairName = aws.ToString(rins.KeyName)
	if rins.Monitoring != nil {
		ins.Monitoring = string(rins.Monitoring.State)
	}
	if rins.Placement != nil {
		ins.ZoneId = aws.ToString(rins.Placement.AvailabilityZone)
	}
	ins.InstanceId = aws.ToString(rins.InstanceId)
	ins.StatusEn = string(rins.State.Name)
	ins.StatusZh = InstanceStateMap[ins.StatusEn]
	if rins.StateReason != nil {
		ins.StatusReason = aws.ToString(rins.StateReason.Message)
	}
	ins.KernelId = aws.ToString(rins.KernelId)
	if len(rins.NetworkInterfaces) > 0 {
		ins.NetworkInterfaces = nil
		for _, iface := range rins.NetworkInterfaces {
			ins.NetworkInterfaces = append(ins.NetworkInterfaces, NetworkInterface{
				PrivateIpAddress:   aws.ToString(iface.PrivateIpAddress),
				NetworkInterfaceId: aws.ToString(iface.NetworkInterfaceId),
				VpcId:              aws.ToString(iface.VpcId),
				SubnetId:           aws.ToString(iface.SubnetId),
				MacAddress:         aws.ToString(iface.MacAddress),
				Status:             string(iface.Status),
				PrivateDnsName:     aws.ToString(iface.PrivateIpAddress),
			})
		}
	}

	if len(rins.Tags) > 0 {
		ins.Tags = nil
		for i := 0; i < len(rins.Tags); i++ {
			if aws.ToString(rins.Tags[i].Key) == "Name" {
				ins.InstanceName = aws.ToString(rins.Tags[i].Key)
			}
			ins.Tags = append(ins.Tags, Tag{
				Key: aws.ToString(rins.Tags[i].Key),
				Value: aws.ToString(rins.Tags[i].Value),
			})
		}
	}

	if len(rins.SecurityGroups) > 0 {
		ins.SecurityGroupList = nil
		for _, sg := range rins.SecurityGroups {
			ins.SecurityGroupList = append(ins.SecurityGroupList, SecurityGroup{
				SecurityGroupId: aws.ToString(sg.GroupId),
				SecurityGroupName: aws.ToString(sg.GroupName),
				VpcId: aws.ToString(rins.VpcId),
			})
		}
	}
	// Eip informer
	ins.EipAddress.IpAddress = aws.ToString(rins.PublicIpAddress)
	ins.EipAddress.DnsName = aws.ToString(rins.PublicDnsName)

	// public-ipv4,
	ins.PublicIpAddress = aws.ToString(rins.PublicIpAddress)
	ins.PublicDnsName = aws.ToString(rins.PublicDnsName)

	// internal-ipv4
	ins.PrivateIpAddress = aws.ToString(rins.PrivateIpAddress)
	ins.PrivateDnsName = aws.ToString(rins.PrivateDnsName)

	//重新生成HostName
	ins.ResetHostName()

}


func (e *Ec2) DescribeInstanceType(insType string) (out string) {
	input := &ec2.DescribeInstanceTypesInput{
		InstanceTypes: []types.InstanceType{types.InstanceType(insType)},
	}
	resp, err := e.Client.DescribeInstanceTypes(context.TODO(), input)
	if err == nil {
		for _, iType := range resp.InstanceTypes {
			if string(iType.InstanceType) == insType {
				vCpu := int(aws.ToInt32(iType.VCpuInfo.DefaultVCpus))
				mem  := int(aws.ToInt64(iType.MemoryInfo.SizeInMiB)) / 1024
				out   = fmt.Sprintf("%dvCPU_%dGiB", vCpu, mem)
				break
			}
		}
	}
	return
}

func (e *Ec2) AssociateEipAddress(instance *Instance) error {
	fmt.Println("---AssociateEipAddress start...")
	//保证未被使用，VPC下的Eip自动允许争夺,Classical网络的受AllowReassociation控制
	_, err := e.GetEipState(instance.EipAddress.AllocationId)
	if err != nil {
		instance.EipAddress.Reason = err.Error()
		return err
	}

	input := &ec2.AssociateAddressInput{
		AllocationId:       aws.String(instance.EipAddress.AllocationId),
		AllowReassociation: false,
		InstanceId:         aws.String(instance.InstanceId),
		PrivateIpAddress:   aws.String(instance.PrivateIpAddress),
	}

	resp, err := e.Client.AssociateAddress(context.TODO(), input)
	if err != nil {
		instance.EipAddress.Reason = err.Error()
	} else {
		instance.EipAddress.BindFinish = true
	}
	fmt.Println("---AssociateEipAddress finish...")
	//_ = resp
	b, _ := json.Marshal(resp)
	fmt.Println(string(b))
	return err
}

func (e *Ec2) GetInstanceByIp(ip, region string) (attr map[string]string, err error) {
	var instance *types.Instance
	var instanceData *InstanceData
	instanceData, err = e.GetInstanceByIps([]string{ip})
	if err != nil {
		return
	}
	if len(instanceData.Instances) > 0 {
		instance = &instanceData.Instances[0]
	} else {
		err = fmt.Errorf("Not found aws instance by ip[%s]", ip)
		return
	}
	ourInstance := Instance{}
	ourInstance.Format(instance)
	attr = map[string]string{
		"vendor": "aws",
		"region": region,
		"id": ourInstance.InstanceId,
		"ip": ourInstance.PrivateIpAddress,
		"public_ipv4": ourInstance.PublicIpAddress,
		"family": ourInstance.InstanceType,
	}
	if len(instance.Tags) > 0 {
		instance.Tags = nil
		for i := 0; i < len(instance.Tags); i++ {
			if aws.ToString(instance.Tags[i].Key) == "Name" {
				attr["name"]= aws.ToString(instance.Tags[i].Key)
			}
		}
	}
	attr["resource"]=e.DescribeInstanceType(ourInstance.InstanceType)
	return
}

func (e *Ec2) UpdateInstanceByIp(ip, hostname string) (err error) {
	var instance *types.Instance
	var instanceData *InstanceData
    instanceData, err = e.GetInstanceByIps([]string{ip})
    if err != nil {
    	return
	}
	if len(instanceData.Instances) > 0 {
		instance = &instanceData.Instances[0]
	} else {
		err = fmt.Errorf("Not found aws instance by ip[%s]", ip)
		return
	}
	input := &ec2.CreateTagsInput{
		Resources: []string{aws.ToString(instance.InstanceId)},
		Tags: []types.Tag{
			{
				Key: aws.String("Name"),
				Value: aws.String(hostname),
			},
		},
	}
	_, err = e.Client.CreateTags(context.TODO(), input)

	if err != nil {
		fmt.Printf("Modify instance hostname [%s] failed: %s\n", hostname, err.Error())
	}
	return
}

//TODO：更新更多内容
func (e *Ec2) UpdateInstance(instance *Instance) bool {
	fmt.Printf("---Start update instance attr <%s>\n", instance.HostName)
	if instance.HostName == "" {
		fmt.Printf("Ignore this tag Name=<%s> , empty\n", instance.HostName)
		return false
	}
	input := &ec2.CreateTagsInput{
		Resources: []string{instance.InstanceId},
		Tags: []types.Tag{
			{
		        Key: aws.String("Name"),
		        Value: aws.String(instance.HostName),
	        },
		},
	}
	_, err := e.Client.CreateTags(context.TODO(), input)

	if err != nil {
		fmt.Printf("Modify instance hostname [%s] failed: %s\n", instance.HostName, err.Error())
		return false
	}
	//修改Name标签值为正确
	for i:= 0; i < len(instance.Tags); i++ {
		if instance.Tags[i].Key == "Name" {
			instance.Tags[i].Value = instance.HostName
		}
	}
	fmt.Println("---Finish update instance Attr")
	return true
}

type CloudInstanceData struct {
	TotalCount int        `json:"TotalCount"`
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

func addInstance(response *ec2.DescribeInstancesOutput, data *CloudInstanceData) {
	for _, reservations := range response.Reservations {
		for _, instance := range reservations.Instances {
			var name string
			for _, tag := range instance.Tags {
				if aws.ToString(tag.Key) == "Name" {
					name = aws.ToString(tag.Value)
					break
				}
			}
			data.Instances = append(data.Instances, CloudInstance{
				Name: name,
				ImageId: aws.ToString(instance.ImageId),
				PrivateIpv4: aws.ToString(instance.PrivateIpAddress),
				InstanceId: aws.ToString(instance.InstanceId),
				Status: string(instance.State.Name),
			})
		}
	}
}

func (e *Ec2) ListInstances(instanceId, pageSize, pageNumber string) (interface{}, error) {
	input := &ec2.DescribeInstancesInput{}
	if instanceId != "" {
		input.InstanceIds = []string{instanceId}
	}
	cloudInstanceData := CloudInstanceData{}
	resp, err := e.Client.DescribeInstances(context.TODO(), input)
	addInstance(resp, &cloudInstanceData)
	cloudInstanceData.TotalCount = len(cloudInstanceData.Instances)
	return cloudInstanceData, err
}

func (e *Ec2) DescribeInstanceByIps(ips []string) (interface{}, error) {
	input := &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name: aws.String("private-ip-address"),
				Values: ips,
			},
		},
	}
	resp, err := e.Client.DescribeInstances(context.TODO(), input)
	return resp, err
}

func (e *Ec2) GetInstanceByIps(ips []string) (*InstanceData, error) {
	resp, err := e.DescribeInstanceByIps(ips)
	var instances InstanceData
	if err == nil {
		response := resp.(*ec2.DescribeInstancesOutput)
		for _, rev := range response.Reservations {
			for _, ins := range rev.Instances {
				instances.Instances = append(instances.Instances, ins)
			}
		}
		if len(instances.Instances) <= 0 {
			err = errors.New(fmt.Sprintf("Instance <%v> not found.", ips))
		}
	}
	return &instances, err
}

func (e *Ec2) DescribeInstance(instanceIds []string) (interface{}, error) {
	input := &ec2.DescribeInstancesInput{InstanceIds: instanceIds}
	resp, err := e.Client.DescribeInstances(context.TODO(), input)
	return resp, err
}

func (e *Ec2) GetInstance(instanceIds []string) (*InstanceData, error) {
	resp, err := e.DescribeInstance(instanceIds)
	var instances InstanceData
	if err == nil {
		response := resp.(*ec2.DescribeInstancesOutput)
		for _, rev := range response.Reservations {
			for _, ins := range rev.Instances {
				instances.Instances = append(instances.Instances, ins)
			}
		}
		if len(instances.Instances) <= 0 {
			err = errors.New(fmt.Sprintf("Instance <%v> not found.", instanceIds))
		}
	}
	return &instances, err
}



func (ins *Instance) ToJson() {
	b, _ := json.Marshal(ins)
	fmt.Println("------json-instance---")
	fmt.Println(string(b))
	fmt.Println("------json-iend---")
}

func (e *Ec2) StartWatch(instances []*Instance, clone *Clone) {
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
		instanceData, err := e.GetInstance(instanceCallback.instanceIds)
		if err != nil {
			message = fmt.Sprintf("Get Instance %v state err: %s\n", instanceCallback.instanceIds, err.Error())
			instanceCallback.SetFinish(message)
			fmt.Println(message)
			return
		}
		if len(instanceData.Instances) <= 0 {
			message = fmt.Sprintf("[%d] Get instance %v state empty: try again, waitting 3s ...\n", i, instanceCallback.instanceIds)
			fmt.Println(message)
			time.Sleep(time.Duration(3)*time.Second)
			continue
		}
		//列表对应关系,更新机器状态
		for _, ins := range instanceData.Instances {
			insId := aws.ToString(ins.InstanceId)
			if _, ok := instanceCallback.instanceMap[insId]; ok {
				instanceCallback.instanceMap[insId].Format(&ins)
				fmt.Printf("%s state=%s\n", instanceCallback.instanceMap[insId].PrivateIpAddress, instanceCallback.instanceMap[insId].StatusEn)
				if instanceCallback.instanceMap[insId].StatusEn == "running" {
					readyCount += 1
				}
			}
		}
		instanceCallback.SetStep(CREATE, -1, "")

		//公网IP刚刚绑定，需要多循环一次
		var again bool
		for _, ins := range instanceCallback.Instances {
			if ins.eipFlag && ins.StatusEn == "running" {
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
	//TODO:逻辑漏洞，instance.StatusEn始终有值
	fmt.Println("Stop watch instance, waiting init.sh exec 180s...")
	//统一等待30s，等待脚本执行完成
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
		if ins.StatusEn == "running" {
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


func (e *Ec2) RegisterJms(insCallback *InstanceCallback, index int) {
	instance := insCallback.Instances[index]
	insCallback.SetStep(SYNCJUMP, index, "")
	fmt.Printf("Host[%s]<%s> Start Jms Register...\n", instance.HostName,instance.PrivateIpAddress)
	n9eCli := n9e.NewN9EClient(config.G.N9eInfo, config.DEFAULT)
	for i:= 0; i < len(instance.JmsBound); i++ {
		//TODO: jump server
		_, err := n9eCli.RegisterJms(instance.PrivateIpAddress,instance.HostName, "aws", []int{instance.JmsBound[i].nid})
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
		resp, err = jmsCli.ExecCommand([]string{host.Id}, "cat /tmp/aws-init10.log")
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

func (e *Ec2) RegisterN9e(insCallback *InstanceCallback, index int) {
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
	insCallback.SetStep(REGISTER, index, "")
	//instance.SetStep(REGISTER)
	host, err := n9eCli.RegisterHost(hostForm)
	if err != nil {
		message = fmt.Sprintf("Host [%s]<%s> register fail: %s\n", instance.PrivateIpAddress, instance.InstanceId, err.Error())
		insCallback.SetStep(FINISH, index, message)
		//instance.SetFinish(message)
		fmt.Println(message)
		return
	}

	fmt.Printf("Host [%s]<%s> register success...\n", host.IP, instance.InstanceId)
	//TODO: 租户填入配置文件
	resp := host.SetHostTenant(config.MAJOR, true, n9eCli)
	if resp.Success {
		// 完成所有
		fmt.Printf("Host [%s]<%s> set tenant success...\n",  host.IP, instance.InstanceId)
		//instance.SetStep(BIND)
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
				})
			}
		}

	} else {
		message = fmt.Sprintf("Host [%s]<%s> set tenant fail: %s\n",  host.IP, instance.InstanceId, resp.Err)
		insCallback.SetStep(FINISH, index, message)
		fmt.Println(message)
	}
}

// 删除
/*
func (e *Ec2)TerminateInstances(instanceIds []string) (*ec2.TerminateInstancesOutput, error) {
	input := &ec2.TerminateInstancesInput{
		InstanceIds: instanceIds,
	}
	resp, err := e.Client.TerminateInstances(context.TODO(), input)
	return resp, err
}
*/


func (e *Ec2)GetEipState(allocationId string) (bool, error) {
	input := &ec2.DescribeAddressesInput{
		Filters: []types.Filter{
			{
				Name: aws.String("allocation-id"),
				Values: []string{allocationId},
			},
		},
	}
	var available bool
	var err       error
	resp, err := e.Client.DescribeAddresses(context.TODO(), input)
	if err == nil {
		if resp.Addresses != nil && len(resp.Addresses) > 0 {
			if resp.Addresses[0].InstanceId != nil {
				err = errors.New(fmt.Sprintf("Eip allocationId <%s>  already in-use with Ec2[%s].", allocationId, aws.ToString(resp.Addresses[0].InstanceId)))
			} else {
				available = true
			}
		} else {
			err = errors.New(fmt.Sprintf("Eip allocationId <%s>  not found.", allocationId))
		}
	}
	fmt.Printf("Eip是否可用: %t\n", available)
	return available, err

}

type TerminateRequest struct{
    InstanceIps []string  `json:"InstanceIps"`
    AccessKey   string    `json:"AccessKey"`
    SecretKey   string    `json:"SecretKey"`
    FormId      int       `json:"FormId"`
	DryRun      bool      `json:"DryRun"`
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
	InstanceId      string `json:"InstanceId"`
	PrivateIpv4     string `json:"PrivateIpv4"`
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

func (e *Ec2)OfflineFromN9e(tResponse *TerminateResponse) {
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

func (e *Ec2)WatchInstanceState(tResponse *TerminateResponse) {
	fmt.Println("Start watch terminate ec2 state...")
	tResponse.Informer()
	var instanceState map[string]string
	filterInput := &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name: aws.String("instance-id"),
				Values: tResponse.InstanceIds,
			},
		},
	}
	watchTime := 60
	var hasOk int
	for i := 0; i <= watchTime; i++ {
		fmt.Printf("[%d] Goroutine filter watch...\n", i+1)
		instanceState = map[string]string{}
		resp, err := e.Client.DescribeInstances(context.TODO(), filterInput)

		if err != nil {
			tResponse.SetFinish(err.Error())
			return
		} else {
			for _, ins := range resp.Reservations[0].Instances {
				if ins.State != nil {
					instanceState[aws.ToString(ins.InstanceId)] = string(ins.State.Name)
				}
			}
		}
		// 请空，分析并更新状态
		hasOk = 0
		for i:= 0; i < len(tResponse.TerminateInstances); i++ {
			if tResponse.TerminateInstances[i].CurrentStateEn == "terminated" {
				hasOk++
				continue
			}
			instanceId := tResponse.TerminateInstances[i].InstanceId
			state, ok := instanceState[instanceId]
			if ok {
				tResponse.TerminateInstances[i].CurrentStateEn = state
				tResponse.TerminateInstances[i].CurrentStateZh = InstanceStateMap[state]
				if state == "terminated" {
					hasOk++
				}
			} else {
				//不存在，认为已经释放删除
				tResponse.TerminateInstances[i].CurrentStateEn = "terminated"
				tResponse.TerminateInstances[i].CurrentStateZh = "已释放"
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

func (e *Ec2)AuthToTerminateInstances(param interface{}, region string) (interface{}, error) {
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

	fmt.Println(tRequest.InstanceIps)
	filterInput := &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name: aws.String("private-ip-address"),
				Values: tRequest.InstanceIps,
			},
		},
	}
	resp, err := e.Client.DescribeInstances(context.TODO(), filterInput)
	if err != nil {
		terminateResponse.SetFinish(err.Error())
		return nil, err
	}
	var instanceIds []string
	instanceIdMap := map[string]string{}
	for _, item := range resp.Reservations {
		for _, instance := range item.Instances {
			id := aws.ToString(instance.InstanceId)
			ip := aws.ToString(instance.PrivateIpAddress)
			var state string
			if instance.State != nil {
				state = string(instance.State.Name)
			}
			instanceIds = append(instanceIds, id)
			instanceIdMap[id] = ip
			terminateResponse.TerminateInstances = append(terminateResponse.TerminateInstances, &TerminateInstanceResult{
				InstanceId:     id,
				PrivateIpv4:    ip,
				CurrentStateEn: state,
				CurrentStateZh: InstanceStateMap[state],
			})
		}
	}
	terminateResponse.InstanceIds = instanceIds

	fmt.Println(instanceIdMap)
	if len(instanceIds) != len(tRequest.InstanceIps) {
		err = errors.New("Filter instance by ip, result count not equal ip.")
		terminateResponse.SetFinish(err.Error())
		return nil, err
	}
	//重新认证客户端
	//TODO
	credential := credentials.NewStaticCredentialsProvider(tRequest.AccessKey, tRequest.SecretKey, "")
	cfg := aws.Config{
		Region: region,
		Credentials: credential,
	}
	ec2Cli := ec2.NewFromConfig(cfg)

	input := &ec2.TerminateInstancesInput{
		InstanceIds: instanceIds,
		DryRun:      tRequest.DryRun,
	}
	fmt.Println("------TResponse---")
	b, _ := json.Marshal(terminateResponse)
	fmt.Println(string(b))
	fmt.Println("------End---")
	response, err := ec2Cli.TerminateInstances(context.TODO(), input)
	if err != nil {
		//dryFlag := "Request would have succeeded, but DryRun flag is set."
		if tRequest.DryRun && strings.Contains(err.Error(), DRYFLAG) {
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
	return response, err

}