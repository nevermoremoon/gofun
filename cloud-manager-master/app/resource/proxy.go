package resource

import (
	"cloud-manager/app/config"
	"cloud-manager/app/modules/aliyun"
	"cloud-manager/app/modules/aws"
	"cloud-manager/app/util/response"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"strings"
)

type CloudVpc interface {
	ListVpcs(pageSize, pageNumber string) (interface{}, error)
	ListSubnets(vpcId, pageSize, pageNumber string) (interface{}, error)
	ListPublicIpv4s(pageSize, pageNumber string) (interface{}, error)
}

type CloudK8S interface {
	NodeGroupScale(cluster, group, service string, number int32) (error)
}

type CloudEcs interface {
	ListInstances(instanceId, pageSize, pageNumber string) (interface{}, error)
	ListDisks(instanceId, pageSize, pageNumber string) (interface{}, error)
	ListInstanceTypes() (interface{}, error)
	ListInstanceTypeFamilies() (interface{}, error)
	ListImages(imageId, pageSize, pageNumber string) (interface{}, error)
	ListKeyPairs(pageSize, pageNumber string) (interface{}, error)
	ListSecurityGroups(vpcId, pageSize, pageNumber string) (interface{}, error)
	CreateInstance(param interface{}) (interface{}, error)
	DescribeInstance(instanceIds []string) (interface{}, error)
	AuthToTerminateInstances(param interface{}, region string) (interface{}, error)
}
type CloudKubernetes interface {
	NodeGroupScale(clusterName, groupName string, number int)
}

type CloudGroup interface {
	ListGroups() (interface{}, error)
}

var (
	Success                = 200
	CreateSuccess          = 201
	RequestFailCode        = 205
	ErrorCloudUnauthorized = 403
	ErrorCloudRequest      = 572
	BadRequest             = 400
	RequestFailed          = 500
)

type CloudProxy struct {
	Vpc   CloudVpc
	Ecs   CloudEcs
	K8S   CloudK8S
	Group CloudGroup
	Name  string
}

/*
func (v CloudProxy) ListVpcs() (interface{}, error) {
	return v.Vpc.ListVpcs()
}

func (v CloudProxy) ListInstanceTypes() (interface{}, error) {
	return v.Ecs.ListInstanceTypes()
}
*/

func checkParam(region, cloud, owner string) (err error) {
	if region == ""  {
		err = fmt.Errorf("Param region is not support, empty.")
	}
	if cloud == "" {
		err = fmt.Errorf("Param cloud is not support, empty.")
	}
	if owner == "" {
		err = fmt.Errorf("Param owner is not support, empty.")
	}
	return
}

func NewCloud(c *gin.Context) (cp CloudProxy, err error) {
	cloud  := c.Query("cloud")
	region := c.Query("region")
	owner  := c.Query("owner")
	err = checkParam(region, cloud, owner)
	if err != nil {
		return
	}
	switch cloud {
	case "aliyun":
		var vpcClient   *aliyun.Vpc
		var ecsClient   *aliyun.Ecs
		var groupClient *aliyun.Group
		var ackClient   *aliyun.ACK
		if auth, ok := config.G.AliyunMap[owner]; ok {
			vpcClient, err   = aliyun.NewVpc(region, auth.AccessKeyId, auth.AccessKeySecret)
			ecsClient, err   = aliyun.NewEcs(region, auth.AccessKeyId, auth.AccessKeySecret)
			groupClient, err = aliyun.NewGroup(region, auth.AccessKeyId, auth.AccessKeySecret)
			ackClient, err   = aliyun.NewACK(region, auth.AccessKeyId, auth.AccessKeySecret)
		} else {
			err = fmt.Errorf("Aliyun not found owner[%s] info.", owner)
		}
		cp = CloudProxy{Vpc: vpcClient, Ecs: ecsClient, Group: groupClient, K8S: ackClient, Name: "aliyun"}
	case "aws":
		var awsEc2Client *ec2.Client
		var awsEksClient *eks.Client
		var ec2Client *aws.Ec2
		var vpcClient *aws.Vpc
		var eksClient *aws.EKS
		if auth, ok := config.G.AwsMap[owner]; ok {
			awsEc2Client, awsEksClient = aws.NewAwsClient(region, auth.AccessKeyId, auth.AccessKeySecret)
			ec2Client, err = aws.NewEc2(awsEc2Client)
			vpcClient, err = aws.NewVpc(awsEc2Client)
			eksClient, err = aws.NewEKS(awsEksClient)
		} else {
			err = fmt.Errorf("aws not found owner[%s] info", owner)
		}
		cp = CloudProxy{Vpc: vpcClient, Ecs: ec2Client, K8S: eksClient, Name:"aws"}
	case "":
		cp, err = CloudProxy{}, fmt.Errorf("no cloud vendors found")
	default:
		cp, err = CloudProxy{}, fmt.Errorf("not Support this cloud vendors")
	}
	return
}

func UpdateInstance(ip, hostname string)(err error) {
	var owner string
	if strings.HasPrefix(ip, "10.70") {
		var ecsClient *aliyun.Ecs
		owner = "1218681829964464"
		if auth, ok := config.G.AliyunMap[owner]; ok {
			ecsClient, err   = aliyun.NewEcs("cn-beijing", auth.AccessKeyId, auth.AccessKeySecret)
		} else {
			err = fmt.Errorf("Aliyun not found owner[%s] info.", owner)
		}
		if err == nil {
			err = ecsClient.UpdateInstanceByIp(ip, hostname)
		}
	} else if strings.HasPrefix(ip, "10.90") {
		var ec2Client *aws.Ec2
		owner = "3891-2558-1212"
		if auth, ok := config.G.AwsMap[owner]; ok {
			awsClient, _ := aws.NewAwsClient("cn-north-1", auth.AccessKeyId, auth.AccessKeySecret)
			ec2Client, err = aws.NewEc2(awsClient)
		} else {
			err = fmt.Errorf("Aws not found owner[%s] info.", owner)
		}
		if err == nil {
			err = ec2Client.UpdateInstanceByIp(ip, hostname)
		}
	} else {
		err = fmt.Errorf("[%s] not unknown vendor", ip)
	}
	return
}

func GetInstanceAttr(ip string)(attr map[string]string, err error) {
	var owner string
	if strings.HasPrefix(ip, "10.70") {
		var ecsClient *aliyun.Ecs
		owner = "1218681829964464"
		if auth, ok := config.G.AliyunMap[owner]; ok {
			ecsClient, err   = aliyun.NewEcs("cn-beijing", auth.AccessKeyId, auth.AccessKeySecret)
		} else {
			err = fmt.Errorf("Aliyun not found owner[%s] info", owner)
		}
		if err == nil {
			attr, err = ecsClient.GetInstanceByIp(ip)
		}
	} else if strings.HasPrefix(ip, "10.90") {
		var ec2Client *aws.Ec2
		owner = "3891-2558-1212"
		if auth, ok := config.G.AwsMap[owner]; ok {
			awsClient, _ := aws.NewAwsClient("cn-north-1", auth.AccessKeyId, auth.AccessKeySecret)
			ec2Client, err = aws.NewEc2(awsClient)
		} else {
			err = fmt.Errorf("Aws not found owner[%s] info.", owner)
		}
		if err == nil {
			attr, err = ec2Client.GetInstanceByIp(ip, "cn-north-1")
		}
	} else {
		err = fmt.Errorf("[%s] not unknown vendor", ip)
	}
	return
}

func ListVpcs(c *gin.Context) {
	utilGin := response.Gin{Ctx: c}
	cloud, err := NewCloud(c)
	pageSize := c.DefaultQuery("PageSize", "10")
	pageNumber := c.DefaultQuery("PageNumber", "1")
	if err != nil {
		utilGin.Response(ErrorCloudUnauthorized, err.Error(), nil)
		return
	}
	data, err := cloud.Vpc.ListVpcs(pageSize, pageNumber)
	if err != nil {
		utilGin.Response(ErrorCloudRequest, err.Error(), nil)
		return
	}
	utilGin.Response(Success, "", data)
}

func ListInstances(c *gin.Context) {
	utilGin := response.Gin{Ctx: c}
	cloud, err := NewCloud(c)
	instanceId := c.Query("InstanceId")
	pageSize := c.DefaultQuery("PageSize", "10")
	pageNumber := c.DefaultQuery("PageNumber", "1")
	if err != nil {
		utilGin.Response(ErrorCloudUnauthorized, err.Error(), nil)
		return
	}
	data, err := cloud.Ecs.ListInstances(instanceId, pageSize, pageNumber)
	if err != nil {
		utilGin.Response(ErrorCloudRequest, err.Error(), nil)
		return
	}
	utilGin.Response(Success, "", data)
}

func ListDisks(c *gin.Context) {
	utilGin := response.Gin{Ctx: c}
	cloud, err := NewCloud(c)
	instanceId := c.Query("InstanceId")
	pageSize := c.DefaultQuery("PageSize", "10")
	pageNumber := c.DefaultQuery("PageNumber", "1")
	if err != nil {
		utilGin.Response(ErrorCloudUnauthorized, err.Error(), nil)
		return
	}
	data, err := cloud.Ecs.ListDisks(instanceId, pageSize, pageNumber)
	if err != nil {
		utilGin.Response(ErrorCloudRequest, err.Error(), nil)
		return
	}
	utilGin.Response(Success, "", data)
}

func ListSubnets(c *gin.Context) {
	utilGin := response.Gin{Ctx: c}
	vpcId := c.Param("id")
	cloud, err := NewCloud(c)
	pageSize := c.DefaultQuery("PageSize", "10")
	pageNumber := c.DefaultQuery("PageNumber", "1")

	if err != nil {
		utilGin.Response(ErrorCloudUnauthorized, err.Error(), nil)
		return
	}
	data, err := cloud.Vpc.ListSubnets(vpcId, pageSize, pageNumber)
	if err != nil {
		utilGin.Response(ErrorCloudRequest, err.Error(), nil)
		return
	}
	utilGin.Response(Success, "", data)
}

func ListInstanceTypes(c *gin.Context) {
	utilGin := response.Gin{Ctx: c}
	cloud, err := NewCloud(c)
	//pageSize := c.DefaultQuery("PageSize", "10")
	//pageNumber := c.DefaultQuery("PageNumber", "1")
	if err != nil {
		utilGin.Response(ErrorCloudUnauthorized, err.Error(), nil)
		return
	}
	data, err := cloud.Ecs.ListInstanceTypes()
	if err != nil {
		utilGin.Response(ErrorCloudRequest, err.Error(), nil)
		return
	}
	utilGin.Response(Success, "", data)
}

func ListInstanceTypeFamilies(c *gin.Context) {
	utilGin := response.Gin{Ctx: c}
	cloud, err := NewCloud(c)
	if err != nil {
		utilGin.Response(ErrorCloudUnauthorized, err.Error(), nil)
		return
	}
	data, err := cloud.Ecs.ListInstanceTypeFamilies()
	if err != nil {
		utilGin.Response(ErrorCloudRequest, err.Error(), nil)
		return
	}
	utilGin.Response(Success, "", data)
}

func ListImages(c *gin.Context) {
	utilGin := response.Gin{Ctx: c}
	cloud, err := NewCloud(c)
	pageSize := c.DefaultQuery("PageSize", "10")
	pageNumber := c.DefaultQuery("PageNumber", "1")
	imageId := c.Query("ImageId")

	if err != nil {
		utilGin.Response(ErrorCloudUnauthorized, err.Error(), nil)
		return
	}
	data, err := cloud.Ecs.ListImages(imageId, pageSize, pageNumber)
	if err != nil {
		utilGin.Response(ErrorCloudRequest, err.Error(), nil)
		return
	}
	utilGin.Response(Success, "", data)
}

func ListKeyPairs(c *gin.Context) {
	utilGin := response.Gin{Ctx: c}
	cloud, err := NewCloud(c)
	pageSize := c.DefaultQuery("PageSize", "10")
	pageNumber := c.DefaultQuery("PageNumber", "1")

	if err != nil {
		utilGin.Response(ErrorCloudUnauthorized, err.Error(), nil)
		return
	}
	data, err := cloud.Ecs.ListKeyPairs(pageSize, pageNumber)
	if err != nil {
		utilGin.Response(ErrorCloudRequest, err.Error(), nil)
		return
	}
	utilGin.Response(Success, "", data)
}

func ListSecurityGroups(c *gin.Context) {
	utilGin := response.Gin{Ctx: c}
	cloud, err := NewCloud(c)
	pageSize := c.DefaultQuery("PageSize", "10")
	pageNumber := c.DefaultQuery("PageNumber", "1")
	vpcId := c.Param("id")

	if err != nil {
		utilGin.Response(ErrorCloudUnauthorized, err.Error(), nil)
		return
	}
	data, err := cloud.Ecs.ListSecurityGroups(vpcId, pageSize, pageNumber)
	if err != nil {
		utilGin.Response(ErrorCloudRequest, err.Error(), nil)
		return
	}
	utilGin.Response(Success, "", data)
}

func ListPublicIpv4s(c *gin.Context) {
	utilGin := response.Gin{Ctx: c}
	cloud, err := NewCloud(c)
	pageSize := c.DefaultQuery("PageSize", "10")
	pageNumber := c.DefaultQuery("PageNumber", "1")

	if err != nil {
		utilGin.Response(ErrorCloudUnauthorized, err.Error(), nil)
		return
	}
	data, err := cloud.Vpc.ListPublicIpv4s(pageSize, pageNumber)
	if err != nil {
		utilGin.Response(ErrorCloudRequest, err.Error(), nil)
		return
	}
	utilGin.Response(Success, "", data)
}

func ListGroups(c *gin.Context) {
	utilGin := response.Gin{Ctx: c}
	cloud, err := NewCloud(c)

	if err != nil {
		utilGin.Response(ErrorCloudUnauthorized, err.Error(), nil)
		return
	}
	data, err := cloud.Group.ListGroups()
	if err != nil {
		utilGin.Response(ErrorCloudRequest, err.Error(), nil)
		return
	}
	utilGin.Response(Success, "", data)
}

//暂时废弃
func DescribeInstance(c *gin.Context) {
	utilGin := response.Gin{Ctx: c}
	cloud, err := NewCloud(c)
	if err != nil {
		utilGin.Response(ErrorCloudUnauthorized, err.Error(), nil)
		return
	}
	instancesQuery := c.Query("Instances")
	instances := strings.Split(instancesQuery, ",")

	data, err := cloud.Ecs.DescribeInstance(instances)
	if err != nil {
		utilGin.Response(ErrorCloudRequest, err.Error(), nil)
		return
	}
	utilGin.Response(Success, "", data)
}

func AuthToTerminateInstances(c *gin.Context) {
	utilGin := response.Gin{Ctx: c}
	region := c.Query("region")
	cloud, err := NewCloud(c)
	if err != nil {
		utilGin.Response(ErrorCloudUnauthorized, err.Error(), nil)
		return
	}
	var data interface{}
	/* 默认只使用json处理，不接受form */
	if cloud.Name == "aliyun" {
		reqBody := aliyun.TerminateRequest{}
		fmt.Println("Gin body参数=======no show")
		err = c.ShouldBindWith(&reqBody, binding.JSON)
		if err == nil {
			data, err = cloud.Ecs.AuthToTerminateInstances(reqBody, region)
		}
	} else if cloud.Name == "aws" {
		fmt.Println("Gin body参数======no show")
		reqBody := aws.TerminateRequest{}
		err = c.ShouldBindWith(&reqBody, binding.JSON)
		if err == nil {
			data, err = cloud.Ecs.AuthToTerminateInstances(reqBody, region)
		}
	}
	if err != nil {
		utilGin.Response(BadRequest, err.Error(), nil)
		return
	}
	utilGin.Response(Success, "", data)
}

func CreateInstance(c *gin.Context) {
	utilGin := response.Gin{Ctx: c}
	cloud, err := NewCloud(c)
	if err != nil {
		utilGin.Response(ErrorCloudUnauthorized, err.Error(), nil)
		return
	}
    var data interface{}
	/* 默认只使用json处理，不接受form */
	if cloud.Name == "aliyun" {
		reqBody := aliyun.InstanceParam{}
		err = c.ShouldBindWith(&reqBody, binding.JSON)
		if err == nil {
			data, err = cloud.Ecs.CreateInstance(reqBody)
		}
	} else if cloud.Name == "aws" {
		reqBody := aws.InstanceParam{}
		err = c.ShouldBindWith(&reqBody, binding.JSON)
		if err == nil {
			data, err = cloud.Ecs.CreateInstance(reqBody)
		}
	}
	fmt.Println("create instance err=", err)
	if err != nil {
		// 特殊定制，更好的选择是BadRequest
		utilGin.Response(RequestFailCode, err.Error(), nil)
		return
	}
	utilGin.Response(CreateSuccess, "", data)
}