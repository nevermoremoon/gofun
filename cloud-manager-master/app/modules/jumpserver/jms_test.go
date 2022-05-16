package jumpserver

import (
	"cloud-manager/app/config"
	"encoding/json"
	"log"
	"time"

	//"encoding/json"
	"fmt"
	"testing"
)

func init() {
	err := config.Parse("/root/goworkspace/src/cloud-manager/config.yml")
	if err != nil {
		log.Fatalln("Parase File error:", err)
	}
}

func NewClient() (client *JmsClient){
	cfg := config.G.JumpInfo
	orgName := "客户监控"
	client =  NewJmsClient(cfg, orgName)
	return
}

func addHost(client *JmsClient) {
	host, err := client.AddHost(RegisterParam{
		Ip:       "3.3.3.3",
		HostName: "linux",
		Nodes:    []string{"af1f200c-cad1-4a3b-8b18-bed1359bc728"},
		Vendor:   "aws",
		Comment:  "This is Linux.",
	})
	if err == nil {
		b, _ := json.Marshal(host)
		fmt.Println("JMS注册成功:", string(b))
	} else {
		fmt.Printf("Jms AddHost fail, err:%s", err.Error())
	}
}

func updateNode(client *JmsClient) {
	host, err := client.NewAssetsHost("10.90.160.186", "hello-echo-160-186")
	if err == nil && host != nil {
		fmt.Println(host.Protocols)
		//host.Labels = []string{} //清除
		//host.Nodes = []string{}
		//b, _ := json.Marshal(host)
		//fmt.Println(string(b))
		fmt.Println("Update: ", host.Update(client))
	} else {
		fmt.Println("Has Error", err)
	}
}


func TestHttp(t *testing.T) {
	cli := NewClient()
	//findHost(cli)
	//addHost(cli)
	//findNode(cli)
	//findChild(cli)
	//createChild(cli)
	//deleteNode(cli)
	//deleteNodeAll(cli)
	//getAssetsNodeByIds(cli)
	//offLine(cli)
	//updateNode(cli)
	//mkdirFullPath(cli)
	//getAllUserGroup(cli)
	//getAllUser(cli)
	//getUser(cli)
	//getUserGroup(cli)
	//syncUser(cli)
	//getExecCommand(cli)
	//execCommand(cli)
	//deleteAllHost(cli)
	//getHostsByIp(cli)
	//exportHost(cli)
	getAllNodes(cli)
}

func exportHost(client *JmsClient) {
	client.ExportHosts("/Users/westos/Desktop")
}

func getAllNodes(client *JmsClient) {
	cli, _ := json.Marshal(*client)
	fmt.Println(string(cli))
	fmt.Println("=========")
	nodes := client.GetAllNodes()
	for _, n := range nodes {
		ss, _ := json.Marshal(n)
		fmt.Println(string(ss))
		break
	}
	fmt.Println("count=", len(nodes))
}

func getHostsByIp(client *JmsClient) {
	hosts, err := client.NewAssetsHostByIP("192.168.200.106")
	if err != nil {
		fmt.Println("NewAssetsHostByIP err:", err)
		return
	}
	b, _ := json.Marshal(hosts)
	fmt.Println(string(b))
}

func deleteAllHost(client *JmsClient) {
	hosts := client.GetAllHosts()
	fmt.Println("len(hosts)=", len(hosts))
	for i := 0; i < len(hosts); i++ {
		err := hosts[i].OffLine(client)
		if err != nil {
			fmt.Printf("%s offline err: %s\n", hosts[i].Ip, err.Error())
		}
	}
}

func getExecCommand(client *JmsClient) {
	resp, err := client.GetExecCommand("9deebe29-1aaa-4f21-981c-4105e781f6c5")
	if err != nil {
		fmt.Println("Has err:", err)
		return
	}
	b, _ := json.Marshal(resp)
	fmt.Println(string(b))
}

func execCommand(client *JmsClient) {
	ids := []string{"2cd2bb3f-6a72-4dd0-b4f8-9ea6cea21d06", "3a6b0909-ca10-419c-8ade-11095c08d7b1"}
	resp, err := client.ExecCommand(ids, "cat /etc/hosts")
	if err != nil {
		fmt.Println("Exec command err:", err)
		return
	}
    taskId := resp.Id
	for {
		res, err := client.GetExecCommand(taskId)
		if err != nil {
			fmt.Println("Get exec command err:", err)
		}
		fmt.Println(">>>>>")
		b, _ := json.Marshal(res)
		fmt.Println(string(b))
		fmt.Println(">>>>>")
		if res.IsFinished {
			break
		}
		fmt.Println("Wait 3s...")
		time.Sleep(time.Duration(3)*time.Second)
	}
	fmt.Println("Exec command finished")
}

func syncUser(client *JmsClient) {
	client.SyncUser()
}

func getUser(client *JmsClient) {
	user, err := client.NewUser("chaoqun.zhai")
	if err != nil {
		fmt.Printf("Has Err: %s\n", err)
		return
	}
	b, _ := json.Marshal(user)
	fmt.Printf(string(b))
}
func getUserGroup(client *JmsClient) {
	userGroup, err := client.NewUserGroup("网络组", "")
	if err != nil {
		fmt.Printf("Has Err: %s\n", err)
		return
	}
	b, _ := json.Marshal(userGroup)
	fmt.Printf(string(b))
}

func getAllUserGroup(client *JmsClient) {
	userGroup, err := client.GetUserGroup()
	if err != nil {
		fmt.Printf("Has Err: %s\n", err)
		return
	}
	b, _ := json.Marshal(userGroup)
	fmt.Printf(string(b))
}

func getAllUser(client *JmsClient) {
	user, err := client.GetUser()
	if err != nil {
		fmt.Printf("Has Err: %s\n", err)
		return
	}
	b, _ := json.Marshal(user)
	fmt.Printf(string(b))
}

func mkdirFullPath(client *JmsClient) {
	path := "/全时云/测试1/测试2/测试3/测试4/测试5"
	node, err := client.NewAssetsNode(path)
	if err != nil {
		fmt.Printf("Has Err: %s\n", err)
	}
	if node == nil {
		fmt.Printf("开始创建全路径: %s", path)
		node = client.MakeJmsFullPath(path)
	}
	if node != nil {
		b, _ := json.Marshal(node)
		fmt.Printf(string(b))
	}
}

// 列出节点
func listNode(client *JmsClient) {
	request := CreateListAssetsNodesRequest()
	resp, err := client.ListAssetsNode(request)
	if err != nil {
		fmt.Println("==", err)
		return
	}

	b, _ := json.Marshal(resp)
	fmt.Println("---------")
	fmt.Println(string(b))
}

// 寻找节点
func findNode(client *JmsClient) {
	node, err := client.NewAssetsNode("/全时云/应用中心/Debug/echo1/hello")
	if err == nil && node != nil {
		_ = node.GetHosts(client)
		b, _ := json.Marshal(node)
		fmt.Println(string(b))
	} else {
		fmt.Println(err)
	}
}

func createChild(client *JmsClient) {
	node, err := client.NewAssetsNode("/全时云/应用中心")
	if err == nil && node != nil {
		child := node.CreateChildren("测试2", client)
		fmt.Println(child)
	} else {
		fmt.Println(err)
	}
}

func findChild(client *JmsClient) {
	node, err := client.NewAssetsNode("/全时云/应用中心")
	if err == nil && node != nil {
		_ = node.GetChild(client)
	} else {
		fmt.Println(err)
	}
}
// 删除节点
func deleteNode(client *JmsClient) {
	node, err := client.NewAssetsNode("/全时云/应用中心/新节点 4")
	if err == nil && node != nil {
		fmt.Println(node.Id)
		fmt.Printf("Delete node [%s] is %v\n", node.Path, node.Delete(client))
	} else {
		fmt.Println("err=", err)
		fmt.Println("node=", node)
	}
}

// 删除所有资产
func deleteNodeAll(client *JmsClient) {
	node, err := client.NewAssetsNode("/全时云/基础组件/容器")
	if err == nil && node != nil {
		fmt.Println(node.Id)
		fmt.Printf("Delete node [%s] is %v\n", node.Path, node.Remove(client))
	} else {
		fmt.Println("err=", err)
		fmt.Println("node=", node)
	}
}

func getAssetsNodeByIds(client *JmsClient) {
	search := []string{"7e56d363-d99c-4677-802d-e60ae6c98873", "67a94fb7-3422-4d73-a1d9-df3798c86bc7"}
	nodes, err := client.GetAssetsNodeByIds(search)
	if err != nil {
		fmt.Println("get Node by Id, has err", err)
	}
	if nodes != nil {
		b, _ := json.Marshal(nodes)
		fmt.Println(string(b))
	}
}


// 下线资产
func offLine(client *JmsClient) {
	host, err := client.NewAssetsHost("1.1.1.1", "www2")
	if err != nil && host != nil {
		fmt.Println(host)
		fmt.Println("OFFLINE: ", host.OffLine(client))
	} else {
		fmt.Println(err)
	}
}

func findHost(client *JmsClient) {
	host, err := client.NewAssetsHost("10.90.32.197", "beta-deb-web-stream-32-197")
	if err == nil && host != nil {
		fmt.Println(host)
		connect, err := host.TestConnect(client)
		if err == nil {
			fmt.Println("connectivity=", connect)
		} else {
			fmt.Println("test err:", err)
		}
		//fmt.Println("OFFLINE: ", host.OffLine())
	} else {
		fmt.Println(err)
	}
}
