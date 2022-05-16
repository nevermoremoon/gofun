package resource

import (
	"cloud-manager/app/config"
	"cloud-manager/app/modules/n9e"
	"cloud-manager/app/util/response"
	"fmt"
	"github.com/gin-gonic/gin"
	"strconv"
	"strings"
	"time"
)

func SyncTree(c *gin.Context) {
	var err error
    var rootNode *n9e.N9ENode
	utilGin := response.Gin{Ctx: c}
	action := c.Query("action")
	nid := c.Query("nid")
	tenant := c.Query("tenant")
	tins, ok := config.G.N9eInfo.TenantMap[tenant]
	if !ok {
		utilGin.Response(BadRequest, fmt.Errorf("tenant <%s> not support", tenant).Error(), nil)
		return
	}
	if nid == "" {
		nid = tins.RootNid
	}
	n9eCli := n9e.NewN9EClient(config.G.N9eInfo, tenant)
	rootNode, err = n9eCli.NewNode(nid)
	if err != nil {
		utilGin.Response(BadRequest, fmt.Errorf("%s> node <%s> find err: %s", tenant, nid, err.Error()).Error(), nil)
		return
	}
	rootNode.SetNamePath(n9eCli)
	fmt.Printf("%s > Action=[%s] start sync... \n", tenant, action)
	switch action {
	case "add":
		go func() {
			startTime := time.Now()
			// 同步从给定的根结点开始，所以所选路径必须在jumpserver中，暂不支持创建当前根路径
			err = n9eCli.SyncAdd(rootNode)
			endTime := time.Now()
			fmt.Printf("-----sync-add-end---------time:%v s\n", endTime.Sub(startTime))
		}()
	case "clean":
		go func() {
			startTime := time.Now()
			//err = n9eCli.SyncClean("/全时云")
			err = n9eCli.SyncClean(rootNode.NamePath)
			if err != nil {
				fmt.Printf("%s> sync-clean err:%v\n", n9eCli.GetOrgName(), err)
			}
			endTime := time.Now()
			fmt.Printf("-----sync-clean-end-------time:%v s\n", endTime.Sub(startTime))
		}()
	default:
		err = fmt.Errorf("action not support")
	}

	if err != nil {
		fmt.Printf("%s> Action=[%s] has err: %s\n", n9eCli.Tenant, action, err.Error())
		utilGin.Response(BadRequest, err.Error(), nil)
		return
	}

	utilGin.Response(Success, "", nil)
}

func SyncVendorInfo(c *gin.Context) {
	var err error
	utilGin := response.Gin{Ctx: c}
	nid := c.Query("nid")
	fmt.Println("---Start update n9e host labels ---")
	if nid == "" {
		err = fmt.Errorf("nid is empty")
		utilGin.Response(BadRequest, err.Error(), nil)
		return
	}
    err = UpdateHostVendorAttr(nid)
	if err != nil {
		utilGin.Response(BadRequest, err.Error(), nil)
		return
	}
	utilGin.Response(Success, "", nil)
}

func GetCdtsPods(n9eCli *n9e.N9EClient) (podsMap map[string]*n9e.N9EPod) {
	podsMap = make(map[string]*n9e.N9EPod)
	//cdts nid
	var n9ePods []*n9e.N9EPod
	var err error
	var cdtsPodNid int

	cdtsPodNid, err = strconv.Atoi(n9eCli.CdtsPodNid)
	if err == nil {
		n9ePods, err = n9eCli.GetNodePods(cdtsPodNid)
	}
	if err != nil {
		fmt.Printf("Get cdts-pod-nid:<%s> cdts pods err: %s\n", n9eCli.CdtsPodNid, err)
		return
	}
	for i:= 0; i < len(n9ePods); i++ {
		if n9ePods[i].Cate != "container" {
			continue
		}
		n9ePods[i].Bind = n9e.SyncBinddins(n9ePods[i].RdbID, n9ePods[i].Ident, n9eCli)
		//fmt.Println(n9ePods[i].Ident, n9ePods[i].Name, n9ePods[i].IP, n9ePods[i].Cate)
		labelMap := n9e.LabelToMap(n9ePods[i].Labels)
		n9ePods[i].LabelMap = labelMap
		if _, ok := labelMap["node_ip"]; ok {
			n9ePods[i].HostIP = labelMap["node_ip"]
			if n9ePods[i].Bind != nil && len(n9ePods[i].Bind.Nodes) > 0 {
				podsMap[labelMap["node_ip"]] = n9ePods[i]
			}
		}
	}
	return
}

func UpdateHostVendorAttr(nid string) (err error) {
	var n9eHosts []*n9e.N9EHost
	nidInt, _ := strconv.Atoi(nid)
	n9eCli := n9e.NewN9EClient(config.G.N9eInfo, config.DEFAULT)
	n9eHosts, err = n9eCli.GetNodeHosts(nidInt)
	if err != nil {
		fmt.Printf("Action=[%s] has err: %s\n", "GetNodeHosts", err.Error())
		return
	}
	cdtsPods := GetCdtsPods(n9eCli)
	fmt.Println("***update label len=", len(n9eHosts))
	for _, host := range n9eHosts {
		if strings.HasPrefix(host.Ident, "10.70") || strings.HasPrefix(host.Ident, "10.90") {
			roleMap := n9e.LabelToMap(host.Labels)
			if _, ok := roleMap["role"]; ok {
				host.Bind = n9e.SyncBinddins(host.RdbID, host.Ident, n9eCli)
				if host.Bind != nil {
					for _, node := range host.Bind.Nodes {
						if strings.HasPrefix(node.Path, "quanshi.basecomp.k8s") {
							attr, rowErr := GetInstanceAttr(host.Ident)
							if rowErr == nil {
								if attr["public_ipv4"] != "" {
									roleMap["eip"] = attr["public_ipv4"]
								} else {
									delete(roleMap, "eip")
								}
								roleMap["eip"] = attr["public_ipv4"]
								roleMap["family"] = attr["family"]
								roleMap["instance"] = attr["id"]
								roleMap["region"] = attr["region"]
								roleMap["resource"] = attr["resource"]
								host.Labels = n9e.LabelToString(roleMap)
								fmt.Printf("update host:%s/%s label success: %v\n", node.Path, host.Ident, host.UpdateLabel(node.Nid, n9eCli))
								//_ = node
								//fmt.Println(host.Ident, host.Labels, host.Bind, attr)
								if strings.Contains(node.Path, "cdts") {
									if _, ok := cdtsPods[host.Ident]; ok {
										pod := cdtsPods[host.Ident]
										pod.LabelMap["public_ip"] =  attr["public_ipv4"]
										pod.Labels = n9e.LabelToString(pod.LabelMap)
										//fmt.Println("***", host.Ident, host.Hostname, pod.Ident, pod.Labels)
										for _, bindNode := range pod.Bind.Nodes {
											if strings.HasPrefix(bindNode.Path, "quanshi.appcenter.tang.cdts") {
												fmt.Printf("** update pod:%s/%s label success: %v\n", bindNode.Path, pod.Ident, pod.UpdateLabel(bindNode.Nid, n9eCli))
											}
										}
									}
								}
							} else {
								fmt.Printf("%s update label has err: %s", host.Ident, rowErr)
							}
							break
						}
					}
				}
			}
		}
	}
	fmt.Println("---End update n9e host labels ---")
	return err
}