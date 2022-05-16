package aws

import (
	"cloud-manager/app/config"
	"cloud-manager/app/modules/console"
	"encoding/json"
	"fmt"
	"log"
	"testing"
	"time"
)

func init() {
	err := config.Parse("/mnt/d/Language/Go/Workspace/src/cloud-manager/config.yml")
	//config.G.InitCloud()
	if err != nil {
		log.Fatalln("Parase File error:", err)
	}
}

func TestAws(t *testing.T) {
	//&Ec2{Client: ec2.NewFromConfig(Cfg)}, Cerr
	owner := "3891-2558-1212"
	auth := config.G.AwsMap[owner]
	awsEC2Client, awsEKSClient := NewAwsClient("cn-north-1", auth.AccessKeyId, auth.AccessKeySecret)
	ec2Client, err := NewEc2(awsEC2Client)
	_ = ec2Client
	eksClient, err := NewEKS(awsEKSClient)
	if err != nil {
		fmt.Println("Has err:", err)
		return
	}
	_ = eksClient
	//deleteEc2(ec2Client)
	//describeEip(ec2Client)
	//associateEip(ec2Client)
	//getExec(ec2Client)
	//testCreateImage(ec2Client)
	//testListImages(ec2Client)
	//testListDisks(ec2Client)
	//testWatchClone(ec2Client)
	//testAutoDeleteImage(ec2Client)
	//testEKSNodeGroup(eksClient)
	//testTryRunClone(ec2Client)
	testGetHostCommand(ec2Client)
}

func testGetHostCommand(cli *Ec2) {
	cmd := getSetHostCommand("xunxun-3-2")
	fmt.Println("===cmd", cmd)
}

func testTryRunClone(cli *Ec2) {
	imageId := cli.getTryCloneImage("ami-05e3b44258d8e752b", "i-0ba4727aa11cdd5b5")
	fmt.Println("===imageId", imageId)
}

func testEKSNodeGroup(cli *EKS) {
	err := cli.NodeGroupScale("beta-eks-tang", "test","aiming.cao", -1)
	if err != nil {
		fmt.Println("Err:", err)
	}
}

func testAutoDeleteImage(cli *Ec2) {
	//success, err := cli.DeleteImage("ami-0a32f6a077a38c25f")
	cli.AutoDeleteCloneImage("ami-0ea5a78e6547a3ac7")

}

func testWatchClone(cli *Ec2) {
	instanceCallback := InstanceCallback{
		Total: 1,
		Result: fmt.Sprintf("0/%d", 1),
		Success: 0,
		IsClone: true,
		callback:  true,
		consoleClient: console.NewConosleClient(config.G.ConsoleInfo),
		FormId: 60,
	}
	imageId, err := cli.watchCloneImage("i-07d562f72d016a3e2", &instanceCallback)
	if err != nil {
		fmt.Println("has err:", err)
		return
	}
	fmt.Println("imageId=", imageId)
}
func testListDisks(cli *Ec2) {
	instanceId := "i-07d562f72d016a3e2"
	resp, err := cli.ListDisks(instanceId, "10", "1")
	if err != nil {
		fmt.Println("list image err:", err)
		return
	}
	b, _ := json.Marshal(resp)
	fmt.Println(string(b))
}

func testListImages(cli *Ec2) {
	imageId := "ami-0a32f6a077a38c25f"
	resp, err := cli.ListImages(imageId, "10", "1")
	if err != nil {
		fmt.Println("list image err:", err)
		return
	}
	imageData := resp.(ImageData)
	var snapshots []string
	for _, disk := range imageData.Images[0].BlockDeviceMapping {
		snapshots = append(snapshots, disk.Ebs.SnapshotId)
	}
	fmt.Println(snapshots)
	resp2, err2 := cli.DescribeSnapshot(snapshots)
	if err2 != nil {
		fmt.Println("list snapshots err:", err)
		return
	}
	b, _ := json.Marshal(resp2)
	fmt.Println(string(b))
}

func testCreateImage(cli *Ec2) {
	imageId, err := cli.CreateImage("i-07d562f72d016a3e2")
	if err != nil {
		fmt.Println("createImage err:", err)
	}
	fmt.Println("imageId=", imageId)
	//imageId := "m-2ze4hrde98wgtq2bel8m"
	if imageId == "" {
		return
	}

	for i := 0; i < 1000; i++ {
		resp, err := cli.ListImages(imageId, "10", "1")
		if err != nil {
			fmt.Println("list image err:", err)
			return
		}
		fmt.Printf("===[%d]\n", i)
		b, _ := json.Marshal(resp)
		fmt.Println(string(b))
		fmt.Println("==EOF")
		time.Sleep(time.Duration(5)*time.Second)
	}
}

func getExec(cli *Ec2) {
	ss:=`{"End":true,"FormId":43,"Instances":[{"Cpu":"2 vCpu","DataDisk":[],"DeletionProtection":false,"Description":"","EipAddress":{"AllocationId":"","BindFinish":false,"IpAddress":"","PublicDnsName":"","Reason":""},"FormId":43,"HostName":"xianxia-e-k8s-3-29","ImageId":"ami-054103975c0f10063","InstanceId":"i-07eac8b2ac8f188a3","InstanceName":"xianxia-e-k8s-3-29","InstanceType":"t3.small","InstanceTypeFamily":"","JmsBound":[{"BindFinish":true,"Path":"","Reason":""},{"BindFinish":true,"Path":"","Reason":""}],"KernelId":"","KeyPairName":"eks-with-k8s","LaunchTime":"2021-05-20 07:38:18 +0000 UTC","Memory":" G","Monitoring":"enabled","OSName":"x86_64","OSType":"xen","PrivateDnsName":"ip-10-90-3-29.cn-north-1.compute.internal","PrivateIpAddress":"10.90.3.29","PublicDnsName":"","PublicIpAddress":"","RegionId":"","SecurityGroupList":[{"Description":"","SecurityGroupId":"sg-033b465e75fd3aa5f","SecurityGroupName":"00-Quanshi-SG","VpcId":"vpc-063fa062"}],"StatusEn":"running","StatusReason":"","StatusZh":"运行中","StepAction":{"CurrentStep":"Finish","Finish":true,"Reason":"","StepEn":"Finish","StepIndex":1000,"StepZh":"完成"}}],"Reason":"","Result":"1/1","Success":1,"Total":1}
`
    insCall := InstanceCallback{}
	err := json.Unmarshal([]byte(ss), &insCall)
	if err != nil {
		fmt.Println("Has Err:", err)
	}
	fmt.Println(insCall)
	cli.RegisterJms(&insCall, 0)
	b, _ := json.Marshal(insCall)
	fmt.Println(string(b))
}

func getInstance() (instance *Instance){
	param := InstanceParam{
		ImageId:         "",
		InstanceType:    "",
		Count:           0,
		Monitoring:      false,
		SubnetId:        "",
		UserData:        "",
		SecurityGroupId: nil,
		KeyPair:         "",
		Disk:            nil,
		Tags:            nil,
		DryRun:          false,
		N9eBound:        nil,
		PublicIpv4:      []string{"eipalloc-594baa64"},
		FormId:          0,
		HostName:        "",
		InstanceName:    "",
	}
	instance = NewEc2Instance(&param, instance.InstanceId, 0)
	instance.PrivateIpAddress = "10.90.33.134"
	return
}

func deleteEc2(cli *Ec2) {
	ipList := []string{"10.90.160.31", "10.90.160.186", "10.90.161.48"}
	resp, err := cli.AuthToTerminateInstances(ipList, "cn-north-1")
	fmt.Println("Error", err)
	fmt.Println(resp)
}

func associateEip(cli *Ec2) {
	instance := getInstance()
	err := cli.AssociateEipAddress(instance)
	if err != nil {
		fmt.Println("AssociateEip err:", err)
	}
}

func describeEip(cli *Ec2) {
	resp, err := cli.GetEipState("eipalloc-07d01dabf1bf86183")
	if err == nil {
		b, _ := json.Marshal(resp)
		fmt.Println(string(b))
	}
}