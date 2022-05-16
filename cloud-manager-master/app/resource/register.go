package resource

import (
	"cloud-manager/app/config"
	"cloud-manager/app/modules/n9e"
	"cloud-manager/app/util/response"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type NormalParam struct {
	IP        string  `json:"ip"`
	Hostname  string  `json:"hostname"`
	Nid       int     `json:"nid"`
	MCNid     int     `json:"mcnid"`
	Vendor    string  `json:"vendor"`
}

func NormalRegister(c *gin.Context) {
	var err    error
	var errMsg string
    var virtualHostname string
	code := Success
	utilGin := response.Gin{Ctx: c}
	fmt.Println("NormalRegister Gin body参数==")
	request:= NormalParam{}
	err = c.ShouldBindWith(&request, binding.JSON)
	if err != nil {
		utilGin.Response(BadRequest, err.Error(), nil)
		return
	}
	b, _ := json.Marshal(request)
	fmt.Println(string(b))
	fmt.Println("===EOF")
	if request.IP == "" || request.Nid == 0 || request.MCNid == 0 {
		utilGin.Response(BadRequest, "param valid, need [ip, nid, mcnid]", nil)
		return
	}
	n9eCli := n9e.NewN9EClient(config.G.N9eInfo, config.DEFAULT)

	go func(ip string, nid int) {
		ec2Err := ModifyInstanceAttr(request.IP, request.Nid, n9eCli)
		if ec2Err != nil {
			fmt.Println(">>>Update vendor's ecs name err:", ec2Err)
		}
	}(request.IP, request.Nid)

	hostForm := n9e.HostRegisterForm{
		IP: request.IP,
		Ident: request.IP,
		Name: request.Hostname,
	}

	host, err := n9eCli.RegisterHost(hostForm)
	if err != nil {
		utilGin.Response(BadRequest, err.Error(), nil)
		return
	}
	fmt.Printf("host:%s register n9e success...\n", host.IP)
	//TODO: 租户填入配置文件
	resp := host.SetHostTenant(config.MAJOR, true, n9eCli)
	if resp.Success {
		fmt.Printf("%s:%d set tenant success\n", host.Ident, request.Nid)
		//TODO: 机器中心, 挂载点
		resp = host.HostBind(request.MCNid, n9eCli)
		if resp.Success {
			fmt.Printf("%s:%d bind machinecenter success\n", host.Ident, request.MCNid)
		}

		resp := host.HostBind(request.Nid, n9eCli)
		//成功一个，将来jms注册一个
		if resp.Success {
			fmt.Printf("%s:%d:%s bind success\n", host.Ident, request.Nid, request.Hostname)
			//register to jumpserver
			virtualHostname, err = n9eCli.RegisterJms(request.IP, request.Hostname, request.Vendor, []int{request.Nid})
			if err != nil {
				errMsg = fmt.Sprintf("Normal node [%s] register failed: %s", request.IP, err.Error())
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
	utilGin.N9eResponse(code, errMsg, virtualHostname)
}

func NormalUnRegister(c *gin.Context) {
	var err error
	utilGin := response.Gin{Ctx: c}
	fmt.Println("NormalUnRegister Gin body参数===")
	request:= NormalParam{}
	err = c.ShouldBindWith(&request, binding.JSON)
	if err != nil {
		utilGin.Response(BadRequest, err.Error(), nil)
		return
	}
	b, _ := json.Marshal(request)
	fmt.Println(string(b))
	fmt.Println("===EOF")
	n9eCli := n9e.NewN9EClient(config.G.N9eInfo, config.DEFAULT)
	host := n9eCli.NewHost(request.IP)
	if host != nil {
		_ = host.Offline(n9eCli)
	}
	_ = n9eCli.OfflineJms(request.IP, request.Hostname)
	utilGin.N9eResponse(Success, "", nil)
}
