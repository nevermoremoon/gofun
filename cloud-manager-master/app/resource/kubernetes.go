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
)

type ApiServerParam struct {
	CName    string `json:"clusterName"`
	Event    string `json:"event"`
	CNid     string `json:"clusterNid"`
	NLBAddr  string `json:"nlbAddr"`
	Command  string
	Hosts    []string
}

func fmtEvent(ctx string) string {
	sliceLine := strings.Split(ctx, "\n")
	for i, s := range sliceLine {
		sliceLine[i] = fmt.Sprintf("# %s", s)
	}
	return strings.Join(sliceLine, "\n")
}

func (ap *ApiServerParam) Initial(n9eCli *n9e.N9EClient) (err error) {
	//scriptUrl := "http://172.17.40.100:8099/packages/cloud-managert/test-apiserver-connect.sh"
	var hosts []*n9e.N9EHost
	scriptUrl := config.G.KubernetesInfo.TestConnectShell
	nidInt, _ := strconv.Atoi(ap.CNid)
	hosts, err = n9eCli.GetNodeHosts(nidInt)
	if err == nil {
		if len(hosts) <= 0 {
			err = fmt.Errorf("The param nid <%s> get empty hosts, please check.", ap.CNid)
		} else {
			ap.Command = fmt.Sprintf(":<<!\n%s\n!\n\ncurl -Ss %s |bash -s -- %s %s 2>&1 |tee /tmp/test-apiserver-connnect-10.log", ap.Event, scriptUrl, ap.CName, ap.NLBAddr)
			for _, h := range hosts {
				ap.Hosts = append(ap.Hosts, h.Ident)
			}
		}
	}
	return
}

func TestApiServerConnect(c *gin.Context) {
	var err error
	var taskId int
	utilGin := response.Gin{Ctx: c}
	fmt.Println("TestApiServerConnect Gin body参数==")
	request := ApiServerParam{}
	n9eCli := n9e.NewN9EClient(config.G.N9eInfo, config.DEFAULT)
	err = c.ShouldBindWith(&request, binding.JSON)
	if err == nil {
		err = request.Initial(n9eCli)
		b, _ := json.Marshal(request)
		fmt.Println(string(b))
		fmt.Println("===EOF")
	}
	if err != nil {
		utilGin.Response(BadRequest, err.Error(), nil)
		return
	}
	title := fmt.Sprintf("test-%s-connect", request.CName)
	taskId, err = n9eCli.RunJobTask(title, "",request.Command, request.Hosts)
	if err != nil {
		utilGin.Response(BadRequest, err.Error(), nil)
		return
	}
	utilGin.Response(Success, "", map[string]interface{}{"task_id": taskId})
}

func NodeGroupScale(c *gin.Context) {
	utilGin := response.Gin{Ctx: c}
	var err error
	reqBody := struct {
		Cluster string `json:"cluster"`
		Service string `json:"service"`
		Group   string `json:"group"`
		Size    int32  `json:"size"`
	}{}

	fmt.Println("Gin body参数=======")
	err = c.ShouldBindWith(&reqBody, binding.JSON)
	if err != nil {
		utilGin.Response(ErrorCloudRequest, err.Error(), nil)
		return
	}
	ss, _ := json.Marshal(reqBody)
	fmt.Println(string(ss))

	cloud, err := NewCloud(c)
	if err == nil {
		err = cloud.K8S.NodeGroupScale(reqBody.Cluster, reqBody.Group, reqBody.Service, reqBody.Size)
	}

	if err != nil {
		utilGin.Response(ErrorCloudRequest, err.Error(), nil)
		return
	}
	utilGin.Response(Success, "scale success", nil)
}
