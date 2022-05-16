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

type FilebeatParam struct {
	InputUrl   string    `json:"input_url"`
	Hosts      []string  `json:"hosts"`
	ConfUrl    string    `json:"conf_url"`
	Action     string    `json:"action"`
	title      string
	args       string
	script     string
}

func (fp *FilebeatParam) Initial() (err error) {
	//scriptUrl := "http://172.17.40.100:8099/packages/filebeat/filebeat.sh"
	scriptUrl := config.G.ConsoleInfo.FilebeatShell
	fp.title = fmt.Sprintf("filebeat-%s", fp.Action)
	switch fp.Action {
	case "install", "restart":
		if fp.ConfUrl == "" || fp.InputUrl == "" {
			err = fmt.Errorf("Action=[%s] need param <conf_url> and <input_url>, but is not all provide.", fp.Action)
		} else {
			fp.args = fmt.Sprintf("-a %s -c %s -i %s", fp.Action, fp.ConfUrl, fp.InputUrl)
		}
		break
	case "reload":
		if fp.InputUrl == "" {
			err = fmt.Errorf("Action=[%s] need param <input_url>, but is not provide.", fp.Action)
		} else {
			fp.args = fmt.Sprintf("-a %s -i %s", fp.Action, fp.InputUrl)
		}
		break
	default:
		err = fmt.Errorf("Action=[%s] don't support.", fp.Action)
	}
	if err == nil {
		if len(fp.Hosts) <= 0 {
			err = fmt.Errorf("The param <hosts> is empty list, provide at least one.")
		} else {
			fp.script = fmt.Sprintf("curl -Ss %s | bash -xs -- %s 2>&1 | tee /tmp/%s.log", scriptUrl, fp.args, fp.Action)
		}
	}
	return
}


func OperateFilebeat(c *gin.Context) {
	var err error
	var taskId int
	utilGin := response.Gin{Ctx: c}
	fmt.Println("OperateFilebeat Gin body参数==")
	request := FilebeatParam{}
	err = c.ShouldBindWith(&request, binding.JSON)
	if err == nil {
		err = request.Initial()
		b, _ := json.Marshal(request)
		fmt.Println(string(b))
		fmt.Println("===EOF")
	}
	if err != nil {
		utilGin.Response(BadRequest, err.Error(), nil)
		return
	}
	n9eCli := n9e.NewN9EClient(config.G.N9eInfo, config.DEFAULT)
	taskId, err = n9eCli.RunJobTask(request.title, "", request.script, request.Hosts)
	if err != nil {
		utilGin.Response(BadRequest, err.Error(), nil)
		return
	}
	utilGin.Response(Success, "", map[string]interface{}{"task_id": taskId})
}

type Host struct {
	Id     int    `json:"id"`
	Ident  string `json:"ident"`
	Ip     string `json:"ip"`
	Name   string `json:"name"`
	Tenant string `json:"tenant"`
	Cate   string `json:"cate"`
	Note   string `json:"note"`
	UUID   string `json:"uuid"`
	Labels string `json:"label"`
}

type Dat struct {
   Nid   int      `json:"nid"`
   Msg   string   `json:"err"`
   Hosts []Host `json:"hosts"`
}

func GetN9eHostsByNid(c *gin.Context) {
	utilGin := response.Gin{Ctx: c}
	nid  := c.Query("nid")
	var dats []Dat

	if nid == "" {
		utilGin.Response(BadRequest, "not found query param: nid", nil)
		return
	}

	n9eCli := n9e.NewN9EClient(config.G.N9eInfo, config.DEFAULT)
	nidSlice := strings.Split(nid, ",")
	for _, n := range nidSlice {
		var dat Dat
		nidInt, _ := strconv.Atoi(n)
		hosts, err := n9eCli.GetNodeHosts(nidInt)
		dat.Nid = nidInt
		dat.Hosts = []Host{}
		if err == nil {
			for _, host := range hosts {
				if host.Cate != "virtual" {
					continue
				}
				dat.Hosts = append(dat.Hosts, Host{
					Id:     host.RdbID,
					Ident:  host.Ident,
					Ip:     host.Ident,
					Name:   host.Hostname,
					Tenant: host.Tenant,
					Cate:   host.Cate,
					Note:   host.Note,
					UUID:   host.UUID,
					Labels: host.Labels,
				})
			}
		} else {
			dat.Msg = err.Error()
		}
		dats= append(dats, dat)
	}
	utilGin.Response(Success, "", dats)
	return
}
