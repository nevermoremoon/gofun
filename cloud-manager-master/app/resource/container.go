package resource

import (
	"cloud-manager/app/config"
	"cloud-manager/app/modules/n9e"
	"cloud-manager/app/util/response"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"strconv"
	"strings"
	"time"
)



type ContainerParam struct {
	IP       string  `json:"ip"`
	Hostname string  `json:"hostname"`
	Nid      int     `json:"nid"`
	Vendor   string  `json:"vendor"`
}

func ModifyInstanceAttr(ip string, nid int, n9eCli *n9e.N9EClient) (err error) {
	var hostname string
	var n9eNode  *n9e.N9ENode

	if strings.HasPrefix(ip, "10.100") {
		fmt.Printf("dexin host <%s> ignore update attr ...\n", ip)
		return
	}

	n9eNode, err = n9eCli.NewNode(strconv.Itoa(nid))

	if err != nil {
		return
	}
	if n9eNode == nil {
		err = fmt.Errorf("[nid:%d] is not found in n9e", nid)
		return
	}
	prefix := n9eNode.GetHostPrefix()
	if prefix == "" {
		err = fmt.Errorf("[%d:%s] get host prefix empty", nid, n9eNode.Path)
		return
	}
	ipSlice := strings.Split(ip, ".")
	ipLen := len(ipSlice)
	hostname = fmt.Sprintf("%s-%s-%s", prefix, ipSlice[ipLen-2], ipSlice[ipLen-1])
	fmt.Println("---Start update instance Name---")
	//等待60s，防止ali instance-name被系统回写。
	time.Sleep(time.Duration(60)*time.Second)
	err = UpdateInstance(ip, hostname)
	fmt.Println("---End update instance Name---")
	return err
}

func ContainerRegister(c *gin.Context) {
	var err    error
	var errMsg string
	code := Success
	utilGin := response.Gin{Ctx: c}
	fmt.Println("ContainerRegister Gin body参数==")
	request:= ContainerParam{}
	err = c.ShouldBindWith(&request, binding.JSON)
	if err != nil {
		utilGin.Response(BadRequest, err.Error(), nil)
		return
	}
	b, _ := json.Marshal(request)
	fmt.Println(string(b))
	fmt.Println("===EOF")
	n9eCli := n9e.NewN9EClient(config.G.N9eInfo, config.DEFAULT)

	ec2Err := ModifyInstanceAttr(request.IP, request.Nid, n9eCli)
	if ec2Err != nil {
		fmt.Println(">>>Update vendor's ecs name err:", ec2Err)
	}
	_, err = n9eCli.RegisterJms(request.IP, request.Hostname, request.Vendor, []int{request.Nid})
	if err != nil {
		errMsg = fmt.Sprintf("container node [%s] register failed: %s", request.IP, err.Error())
		code = RequestFailed
	}
	utilGin.N9eResponse(code, errMsg, nil)
}

func ContainerUnRegister(c *gin.Context) {
	var err error
	utilGin := response.Gin{Ctx: c}
	fmt.Println("ContainerUnRegister Gin body参数===")
	request:= ContainerParam{}
	err = c.ShouldBindWith(&request, binding.JSON)
	if err != nil {
		utilGin.Response(BadRequest, err.Error(), nil)
		return
	}
	b, _ := json.Marshal(request)
	fmt.Println(string(b))
	fmt.Println("===EOF")
	n9eCli := n9e.NewN9EClient(config.G.N9eInfo, config.DEFAULT)
	_ = n9eCli.OfflineJms(request.IP, request.Hostname)
	utilGin.N9eResponse(Success, "", nil)
}
