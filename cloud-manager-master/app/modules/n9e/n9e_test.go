package n9e

import (
	"cloud-manager/app/config"
	"cloud-manager/app/modules/jumpserver"
	"encoding/json"
	_ "encoding/json"
	"fmt"
	"log"
	"testing"
)


func init() {
    err := config.Parse("/Users/westos/Workspace/gopath/src/cloud-manager/config.yml")
    if err != nil {
        log.Fatalln("Parase File error:", err)
    }
}

func NewClient() (client *N9EClient) {
	cfg := config.G.N9eInfo
	client = NewN9EClient(cfg, "customer")
	return
}


func TestHttp(t *testing.T) {
	cli := NewClient()
	//findNode(cli)
	//testRegister(cli)

	//testSyncClean(cli)
	//testSyncAdd(cli)
	//runJobTask(cli)
	//getJobTask(cli)
	//getNodeHosts(cli)
	testRemove(cli)
}

func getNodeHosts(cli *N9EClient) {
	resp, err := cli.GetNodeHosts(177)
	if err != nil {
		fmt.Println("GetNodeHosts err:", err)
	}
	b, _ := json.Marshal(resp)
	fmt.Println(string(b))
	for _, host := range resp {
		fmt.Println(host.Ident, LabelToMap(host.Labels))
	}
}

func getJobTask(cli *N9EClient) {
	resp, err := cli.GetJobTaskDetail(674)
	if err != nil {
		fmt.Println("GetJobTask err:", err)
		return
	}
	b, _ := json.Marshal(resp)
	fmt.Println(string(b))
}

func runJobTask(cli *N9EClient) {
	hosts := []string{"10.90.32.42", "10.70.200.27", "10.70.200.28"}
	taskId, err := cli.RunJobTask("cloud-manager", "", "dat", hosts)
	if err != nil {
		fmt.Println("runJobTask err:", err)
		return
	}
	fmt.Println("Run job success, taskId=", taskId)
}

func testSyncAdd(client *N9EClient) {
	//err := Sync(-1)
	//err := Sync(177)
	root, err := client.NewNode("1x1")
	if err == nil {
		err = client.SyncAdd(root)
	}
	if err != nil {
		fmt.Println(err)
	}
}

func testSyncClean(client *N9EClient) {
	// 删除的前提是与n9e对比后的结果
	err := client.SyncClean("/客户监控/电话客户")
	if err != nil {
		fmt.Println("Sync clean has err:", err)
		return
	}
}

//递归删除某个节点
func testRemove(client *N9EClient) {
    rootPath := "/客户监控/电话客户"
	jmsCli := jumpserver.NewJmsClient(config.G.JumpInfo, client.Tenant)
	root, err := jmsCli.NewAssetsNode(rootPath)
	if err != nil {
		fmt.Printf("remove: get root node=%s fail, err: %v", rootPath, err)
	}
	fmt.Println("remove result: ", root.Remove(jmsCli))
}

func testRegister(client *N9EClient) {
	_, err := client.RegisterJms("2.2.2.2", "hehe", "aws", []int{6702, 6788})
	if err != nil {
		fmt.Println("Register Err:", err)
	}
}

func findNode(cli *N9EClient) {
	node, err := cli.NewNode("4735")
	if err != nil {
		fmt.Println("find err:", err)
	}
	node.SetNamePath(cli)
	if node.IsLeaf() {
		err := node.GetHosts(cli)
		if err != nil {
			fmt.Println("node get hosts err", err)
		}
		b, _ := json.Marshal(node)
		fmt.Println(string(b))
	}
}

/*
func TestSearch(t *testing.T) {
	n9eCli := NewN9EClient(&cfg)
	host := n9eCli.NewHost("10.90.161.48")
	b, _ := json.Marshal(host)
	fmt.Println(string(b))

	err := host.Offline()
	if err != nil {
		fmt.Println(err)
	}
}
*/

/*
func TestNode(t *testing.T) {
	n9eCli := NewN9EClient(config.G.N9eInfo)
	err := n9eCli.RegisterJms("2.2.2.2", "i-eds", "aws", []int{4745, 4750})
	fmt.Println("Has Err:", err)
	//-----
	//node, err := NewNode("439")
	//if err != nil {
	//	fmt.Println(err)
	//}
	//b, _ := json.Marshal(node)
	//fmt.Println(string(b))

	//node.SetNamePath()
}
*/
/*
func TestDelete(t *testing.T) {
	n9eCli := NewN9EClient(config.G.N9eInfo)
	//_ = jumpserver.NewJmsClient(config.G.JumpInfo)
	err := n9eCli.OfflineJms("3.3.3.3", "www-google")
	if err != nil {
		fmt.Println(err)
	}
}
 */

/*
func registerTest() {
	n9eCli := NewN9EClient(config.G.N9eInfo)
	ss := []HostRegisterForm{
		{
			IP: "199.199.199.199",
			Ident: "199.199.199.199",
			Name: "cloud-manager-1",
		},
		{
			IP: "188.188.188.188",
			Ident: "188.188.188.188",
			Name: "cloud-manager-2",
		},
	}

	for _, host := range ss {
		hh, err := n9eCli.RegisterHost(host)
		if  err != nil && hh != nil {
			ok := hh.SetHostTenant("quanshi", true)
			b, _ := json.Marshal(hh)
			fmt.Println("SetTenant", ok)
			fmt.Println(string(b))
		}
	}
}
*/
