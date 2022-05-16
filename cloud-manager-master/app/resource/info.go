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
)

func getN9ENodePathName(nid string, n9eCli *n9e.N9EClient) (hostname, nodePath string, err error) {
	var n9eNode  *n9e.N9ENode
	n9eNode, err = n9eCli.NewNode(nid)
	if err != nil {
		return
	}
	if n9eNode == nil {
		err = fmt.Errorf("[nid:%s] is not found in n9e", nid)
		return
	}
	n9eNode.SetNamePath(n9eCli)
	var prefix string
	prefix, err = n9eNode.GetHostPrefixWithError()
	if err != nil {
        return
	}
	if prefix == "" {
		err = fmt.Errorf("[%s:%s] get host prefix empty", nid, n9eNode.Path)
		return
	}
	//ipSlice := strings.Split(ip, ".")
	//ipLen := len(ipSlice)
	//hostname = fmt.Sprintf("%s-%s-%s", prefix, ipSlice[ipLen-2], ipSlice[ipLen-1])
	hostname = prefix
	nodePath = n9eNode.NamePath
	return hostname, nodePath, nil
}

func GetNodeInfo(c *gin.Context) {
	var err error
	var hostname, nodepath string
	nilMap := map[string]string{}
	utilGin := response.Gin{Ctx: c}
	nid := c.Query("nid")
    fmt.Println("nid=", nid)
	if nid == "" {
		err = fmt.Errorf("nid is empty")
		utilGin.Response(BadRequest, err.Error(), nilMap)
		return
	}
	n9eCli := n9e.NewN9EClient(config.G.N9eInfo, config.DEFAULT)
	hostname, nodepath, err = getN9ENodePathName(nid, n9eCli)

	if err != nil {
		utilGin.Response(BadRequest, err.Error(), nilMap)
		return
	}
	utilGin.Response(Success, "", map[string]string{"hostname": hostname, "nodepath": nodepath})
}

func HostRegister(c *gin.Context) {
	var err    error
	var errMsg string
	nilMap := map[string]string{}
	code := Success
	utilGin := response.Gin{Ctx: c}

	request:= ContainerParam{}
	err = c.ShouldBindWith(&request, binding.JSON)
	if err != nil {
		utilGin.Response(BadRequest, err.Error(), nilMap)
		return
	}
	fmt.Printf("//-- start register dexin host:%s...\n", request.IP)
	b, _ := json.Marshal(request)
	fmt.Println(string(b))

	n9eCli := n9e.NewN9EClient(config.G.N9eInfo, config.DEFAULT)

	hostForm := n9e.HostRegisterForm{
		IP: request.IP,
		Ident: request.IP,
		Name: request.Hostname,
	}

	host, err := n9eCli.RegisterHost(hostForm)
	if err != nil {
		utilGin.Response(BadRequest, err.Error(), nilMap)
		return
	}
	fmt.Printf("host:%s register n9e success...\n", host.IP)
	//TODO: 租户填入配置文件
	resp := host.SetHostTenant(config.MAJOR, true, n9eCli)
	if resp.Success {
		fmt.Printf("%s:%d set tenant success\n", host.Ident, request.Nid)
		//TODO: 机器中心, 挂载点
		dexinNid, _ := strconv.Atoi(config.G.N9eInfo.DexinNid)
		resp = host.HostBind(dexinNid, n9eCli)
		if resp.Success {
			fmt.Printf("%s:%d bind success\n", host.Ident, dexinNid)
		}

		resp := host.HostBind(request.Nid, n9eCli)
		//成功一个，将来jms注册一个
		if resp.Success {
			fmt.Printf("%s:%d:%s bind success\n", host.Ident, request.Nid, request.Hostname)
			_, err = n9eCli.RegisterJms(request.IP, request.Hostname, request.Vendor, []int{request.Nid})
			if err != nil {
				errMsg = fmt.Sprintf("dexin host:%s register failed: %s", request.IP, err.Error())
				code = RequestFailed
			}
		} else {
			code = RequestFailed
			errMsg = resp.Err
		}
	} else {
		code = RequestFailed
		errMsg = resp.Err
	}
	fmt.Println("//--EOF")
	utilGin.N9eResponse(code, errMsg, nilMap)
}