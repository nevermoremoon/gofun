package aliyun

import (
	"cloud-manager/app/config"
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

func TestAli(t *testing.T) {
	//&Ec2{Client: ec2.NewFromConfig(Cfg)}, Cerr
	owner := "1218681829964464"
	auth := config.G.AliyunMap[owner]
	ecsClient, err := NewEcs("cn-beijing", auth.AccessKeyId, auth.AccessKeySecret)
	if err != nil {
		fmt.Println("new ecs err:", err)
		return
	}
	ackClient, err := NewACK("cn-beijing", auth.AccessKeyId, auth.AccessKeySecret)
	if err != nil {
		fmt.Println("new ack err:", err)
		return
	}
	_ = ecsClient
	_ = ackClient
	//testCreateImage(ecsClient)
	//testDeleteImage(ecsClient)
	//testAckScale(ackClient)
	//testTryRunClone(ecsClient)
	testGetHostCommand(ecsClient)
}



func testGetHostCommand(cli *Ecs) {
	cmd := getSetHostCommand("xunxun-3-2")
	fmt.Println("===cmd", cmd)
}

func testTryRunClone(cli *Ecs) {
	imageId := cli.getTryCloneImage("m-2ze3duxufawsxomqajgq", "i-2zeahbwsh8vju21v6jy8")
	fmt.Println("===imageId", imageId)
}

func testAckScale(cli *ACK) {
	err := cli.NodeGroupScale("beta-ack-tang", "default", "echo", 1)
	if err != nil {
		fmt.Println("scale fail:", err)
	}
}

func testDeleteImage(cli *Ecs) {
	imageId := "m-2zeauk43zyovql363nfz"
	cli.AutoDeleteCloneImage(imageId)
}

func testCreateImage(cli *Ecs) {
	imageId, err := cli.CreateImage("i-2zeioly8ub9e4dto6yy3")
	if err != nil {
		fmt.Println("createImage err:", err)
	}
	fmt.Println("imageId=", imageId)
	//imageId := "m-2ze4hrde98wgtq2bel8m"

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