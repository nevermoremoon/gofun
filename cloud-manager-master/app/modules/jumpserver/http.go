package jumpserver

import (
    "cloud-manager/app/common/request"
    "cloud-manager/app/config"
    "crypto"
    "encoding/json"
    "errors"
    "fmt"
    "github.com/fatih/structs"
    "github.com/go-fed/httpsig"
    "github.com/wxnacy/wgo/arrays"
    "net/url"

    //uuid "github.com/satori/go.uuid"
    "io/ioutil"
    "net/http"
    "path/filepath"
    "strings"
    "time"
)


func NewJmsClient(obj *config.Jumpserver, orgName string) *JmsClient {
    jmsCli := deepCopy(obj)
    // set other org, else use default
    if orgName == config.MAJOR {
        orgName = config.DEFAULT
    }
    if orgName != "default" && orgName != "" {
        if externalTenant , ok := jmsCli.ExternalTenantMap[orgName]; ok {
            jmsCli.OrgName = externalTenant.OrgName
            jmsCli.OrgId   = externalTenant.OrgId
            jmsCli.AdminUser = externalTenant.AdminUser
            jmsCli.AdminId = externalTenant.AdminId
            jmsCli.SystemUser = externalTenant.SystemUser
            jmsCli.SystemId= externalTenant.SystemId
        }
    }
    return jmsCli
}



func (jmsCli *JmsClient) GetUsers() {
    action := JmsAction{
        Path:         request.JmsRoute["assets-nodes"],
        Method:       request.GET,
    }
    resp := jmsCli.JmsRequest(&action)
    _ = resp
}

type AssetsNodesRequest struct {
    Action     *JmsAction
    Key        string
    Value      string
    Id         string
    Search     string     //模糊匹配
    Order      string
    Spm        string
    Ids        string     //id列表，逗号分割，批量查询
    Limit      int
    Offset     int
}

type AssetsNodesResponse struct {
    Results  []AssetsNode   `json:"results"`
    Previous string         `json:"previous"`
    Next     string         `json:"next"`
    Count    int            `json:"count"`
}


type AssetsNode struct {
    Id      string `json:"id"`
    Key     string `json:"key"`
    Value   string `json:"value"`      //唯一性， 接受更改， 同时更新name，full_value
    OrgId   string `json:"org_id"`
    Name    string `json:"name"`
    Path    string `json:"full_value"`
    OrgName string `json:"org_name"`
    Hosts   []AssetsHost `json:"hosts"`
}

//---
type AssetsHostsRequest struct {
    Action     *JmsAction
    HostName   string
    Id         string
    Ip         string
    Search     string     //模糊匹配
    Order      string
    Spm        string
    Ids        string     //id列表，逗号分割，批量查询
    Ips        string
    Limit      int
    Offset     int
}

type AssetsHostsResponse struct {
    Results  []AssetsHost   `json:"results"`
    Previous string         `json:"previous"`
    Next     string         `json:"next"`
    Count    int            `json:"count"`
}


type Connectivity struct {
    Status   int    `json:"status"`
    Datetime string `json:"datetime"`
}
//-----
// TODO: 更多字段待完善
type AssetsHost  struct {
    Id           string       `json:"id"`
    Hostname     string       `json:"hostname"`
    Ip           string       `json:"ip"`
    Protocols    []string     `json:"protocols"`
    PublicIpv4   string       `json:"public_ip"`
    AdminUser    string       `json:"admin_user"`
    Nodes        []string     `json:"nodes"`
    Platform     string       `json:"platform"`
    Vendor       string       `json:"vendor"`
    NodesPath    []string     `json:"nodes_display"`
    Labels       []string     `json:"labels"`
    Comment      string       `json:"comment"`
    IsActive     bool         `json:"is_active"`
    Connectivity Connectivity `json:"connectivity"`
}

type AssetsHostAddRequest struct {
    Action *JmsAction
    Param  map[string]interface{}
}

func CreateAssetsHostAddRequest() *AssetsHostAddRequest {
    req := AssetsHostAddRequest{
        Action: &JmsAction{
            Path: request.JmsRoute["assets-assets"],
            Method: request.POST,
        },
        Param: map[string]interface{}{
            "protocols": []string{"ssh/22"},
            "admin_user": config.G.JumpInfo.AdminId,
            "platform": "Linux",
        },
    }
    return &req
}

//type AssetsHostResponse AssetsHost
func (jmsCli *JmsClient) AssetsHostAdd(request *AssetsHostAddRequest) (*AssetsHost, error) {
    var err error
    host := AssetsHost{}
    resp := jmsCli.JmsRequest(request.Action)
    if resp.Success {
        err = convertToStruct(resp.Data, &host)
    } else {
        err = errors.New(resp.Err)
    }
    return &host, err
}

type RegisterParam struct {
    Ip       string    `json:"Ip"`
    HostName string    `json:"HostName"`
    Nodes    []string  `json:"Nodes"`
    Vendor   string    `json:"Vendor"`
    Comment  string    `json:"comment"`
}

//---Host

func (jmsCli *JmsClient)NewAssetsHost(ip, hostname string) (*AssetsHost, error) {
    var host *AssetsHost
    var err error
    req := CreateListAssetsHostsRequest()
    req.Ip = ip
    req.HostName = hostname
    resp, err := jmsCli.ListAssetsHost(req)
    b, _ := json.Marshal(resp)
    fmt.Println(string(b))
    if err == nil {
        for _, n := range resp.Results {
            if n.Ip == ip {
                host = &n
                break
            }
        }
    }
    return host, err
}

func (jmsCli *JmsClient)NewAssetsHostByIP(ip string) ([]AssetsHost, error) {
    var hosts []AssetsHost
    var err error
    req := CreateListAssetsHostsRequest()
    req.Ip = ip
    resp, err := jmsCli.ListAssetsHost(req)
    b, _ := json.Marshal(resp)
    fmt.Println(string(b))
    if err == nil {
        hosts = resp.Results
    }
    return hosts, err
}

func CreateListAssetsHostsRequest() *AssetsHostsRequest {
    req := AssetsHostsRequest{
        Limit: 100000,
        Offset: 0,
        Action: &JmsAction{
            Path: request.JmsRoute["assets-assets"],
            Method: request.GET,
        },
    }
    return &req
}

func (jmsCli *JmsClient) ListAssetsHost(request *AssetsHostsRequest) (*AssetsHostsResponse, error) {
    query := buildHttpQuery(structs.Map(request))
    if query != "" {
        request.Action.Path = fmt.Sprintf("%s?%s", request.Action.Path, query)
    }
    fmt.Println(request.Action.Path)
    assetsHostsResponse := AssetsHostsResponse{}
    resp := jmsCli.JmsRequest(request.Action)
    err := convertToStruct(resp.Data, &assetsHostsResponse)
    return &assetsHostsResponse, err
}

// 节点无关性方法， 机器注册时绑定节点
func (jmsCli *JmsClient)AddHost(param RegisterParam) (*AssetsHost, error) {
    var host *AssetsHost
    var err error
    req := CreateAssetsHostAddRequest()
    req.Param["ip"] = param.Ip
    req.Param["hostname"] = param.HostName
    req.Param["nodes"] = param.Nodes
    req.Param["comment"] = param.Comment
    req.Param["vendor"] = param.Vendor
    req.Action.Payload = req.Param
    host, err = jmsCli.AssetsHostAdd(req)
    return host, err
}

// 主机下线
func (host *AssetsHost) OffLine(jmsCli *JmsClient) error {
    var err error
    Action := &JmsAction{
        Path: fmt.Sprintf("%s%s/", request.JmsRoute["assets-assets"], host.Id),
        Method: request.DELETE,
    }
    resp := jmsCli.JmsRequest(Action)
    if resp.Success {
        //err = convertToStruct(resp.Data, &host)
        fmt.Printf("=== Host [%s] is offline form jumper server\n", host.Ip)
    } else {
        fmt.Printf("=== Host [%s] is offline form jumper server err:%s\n", host.Ip, resp.Err)
        err = errors.New(resp.Err)
    }
    return err
}

// 更新主机信息
func (host *AssetsHost) Update(jmsCli *JmsClient) error {
    if host == nil {
        return errors.New("Host is nil.")
    }
    var comment string
    nodes, err := jmsCli.GetAssetsNodeByIds(host.Nodes)
    if err != nil {
        fmt.Printf("%s> Update calculate comment by GetAssetsNodeByIds() err: %s\n", jmsCli.OrgName, err.Error())
    } else {
        for _, node := range nodes {
            if strings.HasPrefix(node.Path, "/全时云/机器中心") {
                continue
            }
            if comment == "" {
                comment = node.Path
            } else {
                comment = fmt.Sprintf("%s, %s", comment, node.Path)
            }
        }
    }
    if comment != "" {
        host.Comment = comment
    }
    action := &JmsAction{
        Path: fmt.Sprintf("%s%s/", request.JmsRoute["assets-assets"], host.Id),
        Method: request.PUT,
        Payload: map[string]interface{}{
            "hostname": host.Hostname,
            "ip": host.Ip,
            "platform": host.Platform,
            "protocols": host.Protocols,
            "nodes": host.Nodes,       //更新字段
            "labels": host.Labels,     //更新字段
            "comment": host.Comment,
        },
    }
    resp := jmsCli.JmsRequest(action)
    return errors.New(resp.Err)
}

// 为需要注册的n9e-host创建全链路路径，解决不实时问题.
func (jmsCli *JmsClient) MakeJmsFullPath(path string) *AssetsNode {
    fmt.Printf("开始创建全路径: %s\n", path)
    path = strings.Trim(path, "/")
    paths := strings.Split(path, "/")
    var currentPath string
    currentPath = ""
    var parentNode *AssetsNode
    for _, p := range paths {
        var err error
        var node *AssetsNode
        currentPath = fmt.Sprintf("%s/%s", currentPath, p)
        node, err = jmsCli.NewAssetsNode(currentPath)
        if err != nil {
            fmt.Printf("%s> RecursiveMakeJmsPath [%s] NewAssetsNode err: %s", jmsCli.OrgName, path, err.Error())
            return nil
        }
        //没找到,创建
        if node == nil {
            if parentNode != nil {
                node = parentNode.CreateChildren(p, jmsCli)
            }
        } else {
            fmt.Printf("[%s] already exists, ignore... \n", node.Path)
        }
        parentNode = node
        if parentNode == nil {
            break
        }
    }
    return parentNode
}


func (host *AssetsHost) IsConnect() bool {
    return host.Connectivity.Status == 1
}

func (host *AssetsHost) TestConnect(jmsCli *JmsClient) (connectivity bool, err error) {
    if host.IsConnect() {
        connectivity = true
        return
    }
    action := &JmsAction{
        Path: strings.Replace(request.JmsRoute["connect-test"], ":id", host.Id,-1),
        Method: request.POST,
        Payload: map[string]interface{}{
            "action":"test",
        },
    }
    resp := jmsCli.JmsRequest(action)
    if !resp.Success {
        err = fmt.Errorf("[%s] test connect fail: %s\n", host.Ip, resp.Err)
        return
    }
    //等待20s,获取结果, 经验值
    //fmt.Printf("[%d] Host [%s] test connect, wait 10s...\n", i, host.Ip)
    time.Sleep(time.Duration(20)*time.Second)
    newHost, err := jmsCli.NewAssetsHost(host.Ip, host.Hostname)
    if err == nil && newHost != nil {
        connectivity = newHost.IsConnect()
    }
    fmt.Println(" connectivity=", connectivity)

    /*
    for i := 0; i < 6; i++ {
        fmt.Printf("[%d] Host [%s] test connect, wait 10s...\n", i, host.Ip)
        time.Sleep(time.Duration(10)*time.Second)
        newHost, err := NewAssetsHost(host.Ip, host.Hostname)
        if err != nil {
            return connectivity, err
        }
        if newHost != nil {
            connectivity = newHost.IsConnect()
            if connectivity {
                break
            }
        }
    }
     */
    return
}

// 从节点上分离， 如果为空，则下线
func (host *AssetsHost) Detach(nodeIds []string, jmsCli *JmsClient) bool {
    var indexedCount int   //被索引的次数
    var reserve []string
    indexedCount = len(host.Nodes)
    for _, node := range host.Nodes {
        //没找到的保留
        if arrays.ContainsString(nodeIds, node) < 0 {
            reserve = append(reserve, node)
        }
    }
    //没有需要保留的，且仅一个索引就是此节点，才直接下线
    if len(reserve) < 1 && indexedCount == 1 {
        //不存在，视为成功
        _ = host.OffLine(jmsCli)
        return true
    }

    // 重置nodes，然后更新
    fmt.Printf("%s>=================开始更新%v ==> %v\n", jmsCli.OrgName, host.Nodes, reserve)
    host.Nodes = reserve
    err := host.Update(jmsCli)

    return err == nil
}



func (jmsCli *JmsClient) GetAssetsNodeByIds(ids []string) ([]AssetsNode, error) {
    var nodes []AssetsNode
    var err error
    //使用value缩小范围，然后利用路径精确匹配
    req := CreateListAssetsNodesRequest()
    req.Ids = strings.Join(ids, ",")
    req.Limit = 100000
    resp, err := jmsCli.ListAssetsNode(req)
    //需在创建时避免重复,value，full_value的组合不能保证唯一性
    if resp != nil {
        nodes = resp.Results
    }
    return nodes, err
}

// -----查找节点
func (jmsCli *JmsClient) NewAssetsNode(path string) (*AssetsNode, error) {
    var node *AssetsNode
    var err error
    name := filepath.Base(path)
    if name == "" {
        return nil, errors.New("Path is empty.")
    }
    //使用value缩小范围，然后利用路径精确匹配
    req := CreateListAssetsNodesRequest()
    req.Value = name
    resp, err := jmsCli.ListAssetsNode(req)
    //需在创建时避免重复,value，full_value的组合不能保证唯一性
    if err == nil {
        for _, n := range resp.Results {
            if n.Path == path {
                node = &n
                break
            }
        }
    }
    return node, err
}


type NodeAssetsRequest struct {
    Action             *JmsAction
    Key                string
    Node_Id            string
    Show_Current_Asset int
    Display            int
    Draw               int
    Limit              int
    Offset             int
}

type NodeAssetsResponse struct {
    Results  []AssetsHost   `json:"results"`
    Previous string         `json:"previous"`
    Next     string         `json:"next"`
    Count    int            `json:"count"`
}


func CreateLisNodeAssetsRequest() *NodeAssetsRequest {
    req := NodeAssetsRequest{
        Limit: 100,
        Offset: 0,
        Display: 1,
        Draw: 1,
        Show_Current_Asset: 1,    //仅显示该节点的资产，不展示子节点的
        Action: &JmsAction{
            Path: request.JmsRoute["assets-assets"],
            Method: request.GET,
        },
    }
    return &req
}

func (node *AssetsNode) appendHosts(nodeAssetsResponse *NodeAssetsResponse) {
    node.Hosts = append(node.Hosts, nodeAssetsResponse.Results...)
}

func (node *AssetsNode) ParentKey() (pkey string) {
    keySlice := strings.Split(node.Key, ":")
    //顶级节点
    if len(keySlice) <= 1 {
        return
    }
    pkey = strings.Join(keySlice[:len(keySlice)-1], ":")
    return
}

func (node *AssetsNode) IsLeaf() bool {
    keySlice := strings.Split(node.Key, ":")
    return len(keySlice) == 5
}

// 获取节点资产
func (node *AssetsNode) GetHosts(jmsCli *JmsClient) error {
    req := CreateLisNodeAssetsRequest()
    req.Node_Id = node.Id
    query := buildHttpQuery(structs.Map(req))
    if query != "" {
        req.Action.Path = fmt.Sprintf("%s?%s", req.Action.Path, query)
    }
    nodeAssetsResponse := NodeAssetsResponse{}
    resp := jmsCli.JmsRequest(req.Action)
    err := convertToStruct(resp.Data, &nodeAssetsResponse)
    if err != nil {
        return err
    }
    node.appendHosts(&nodeAssetsResponse)
    next := nodeAssetsResponse.Next
    for {
        if next == "" {
            break
        } else {
            nar := NodeAssetsResponse{}
            urlObj, err := url.Parse(next)
            if err != nil {
                fmt.Println("Url Parse err:", err)
                break
            }
            req.Action.Path = fmt.Sprintf("%s?%s", urlObj.Path, urlObj.RawQuery)
            resp := jmsCli.JmsRequest(req.Action)
            err = convertToStruct(resp.Data, &nar)
            if err == nil && resp.Success {
                node.appendHosts(&nar)
                next = nar.Next
            } else {
                //失败，会造成next中断
                break
            }
        }
    }
    return err
}

// 创建子节点  Key 0:3:2 才是唯一标示
func (node *AssetsNode) CreateChildren(name string, jmsCli *JmsClient) (child *AssetsNode) {
    childPath := fmt.Sprintf("%s/%s", node.Path, name)
    child, _ = jmsCli.NewAssetsNode(childPath)
    if child != nil {
        fmt.Printf("%s> Node [%s] already exists, ignore...\n", jmsCli.OrgName, child.Path)
        return child
    }
    //初始化实例
    child = &AssetsNode{}
    action := &JmsAction{
        Path: strings.Replace(request.JmsRoute["assets-nodes-children"], ":nid", node.Id, -1),
        Method: request.POST,
        Payload: map[string]interface{}{
            "value": name,
        },
    }
    resp := jmsCli.JmsRequest(action)
    if resp.Success {
        err := convertToStruct(resp.Data, child)
        if err != nil {
            fmt.Println("CreateChildren convertToStruct err:", err)
        }
    }
    return child
}

// 删除节点
func (node *AssetsNode) Delete(jmsCli *JmsClient) bool {
    action := &JmsAction{
        Path: fmt.Sprintf("%s%s/", request.JmsRoute["assets-nodes"], node.Id),
        Method: request.DELETE,
    }
    resp := jmsCli.JmsRequest(action)
    return resp.Success
}

func (jmsCli *JmsClient)GetAllHosts() []*AssetsHost {
    action := &JmsAction{
        Path: request.JmsRoute["assets-assets"],
        Method: request.GET,
    }
    resp := jmsCli.JmsRequest(action)
    var hosts []*AssetsHost
    if resp.Success {
        err := convertToStruct(resp.Data, &hosts)
        if err != nil {
            fmt.Println("GetAllHosts convertToStruct err:", err)
        }
    }
    return hosts
}

func (jmsCli *JmsClient)ExportHosts(filePath string) (file string, err error) {
    hosts := jmsCli.GetAllHosts()
    date := time.Now().Format("2006-01-02")
    file = fmt.Sprintf("%s/jumpserver-%s.xlsx", filePath, date)
    err = Export(file, "jumpserver", hosts)
    if err != nil {
        fmt.Println("Export hosts, err:", err)
    }
    return
}

func (jmsCli *JmsClient)GetAllNodes() (nodes []*AssetsNode) {
    action := &JmsAction{
        Path: request.JmsRoute["assets-nodes"],
        Method: request.GET,
    }
    resp := jmsCli.JmsRequest(action)
    if resp.Success {
        err := convertToStruct(resp.Data, &nodes)
        if err != nil {
            fmt.Println("GetAllNodes convertToStruct err:", err)
        }
    }
    return nodes
}


// 添加资产到节点
func (node *AssetsNode) AddHosts(hostIds []string, jmsCli *JmsClient) bool {
    action := &JmsAction{
        Path: strings.Replace(request.JmsRoute["assets-nodes-add"], ":id", node.Id, -1),
        Method: request.PUT,
        Payload: map[string]interface{}{
            "assets": hostIds,
        },
    }
    resp := jmsCli.JmsRequest(action)
    return resp.Success
}

// 移除资产从节点上,需host自身解除该路径绑定， 如果已经是最后一个路径，则放置到游离区，暂时直接下线
func (node *AssetsNode) RemoveHosts(hostIds []string, all bool, jmsCli *JmsClient) bool {
    err := node.GetHosts(jmsCli)
    if err != nil {
        fmt.Printf("%s> RemoveHosts get node [%s] hosts err: %s\n",jmsCli.OrgName, node.Path, err.Error())
        return false
    } else {
        var hostIps []string
        for _, h := range node.Hosts {
            hostIps = append(hostIps, h.Ip)
        }
        fmt.Println("Get Hosts=", hostIps)
    }

    for _, host := range node.Hosts {
        //分离所有
        if all {
            //fmt.Printf("========== Host [%s] start detach\n", host.Ip)
            host.Detach([]string{node.Id}, jmsCli)
            //fmt.Printf("========== Host [%s] finish detach\n", host.Ip)
            continue
        }
        //如果存在，自己分离
        if arrays.ContainsString(hostIds, host.Id) > -1 {
            host.Detach([]string{node.Id}, jmsCli)
        }
    }
    return true
}

// 删除节点下所有, root节点资产可以删除，本身不可以删除
func (node *AssetsNode) recursiveDelete(jmsCli *JmsClient) bool {
    //fmt.Printf("=== Node [%s] start delete\n", node.Path)
    var isSuccess bool
    children := node.GetChild(jmsCli)
    if len(children) > 0 {
        for _, child := range children {
            isSuccess = child.recursiveDelete(jmsCli)
            if !isSuccess {
                return isSuccess
            }
        }
    }

    // 走到这，已成为叶子节点，开始移除节点下所有资产
    var isOk bool
    //fmt.Printf("======= Node [%s] start delete hosts\n", node.Path)
    isOk = node.RemoveHosts(nil, true, jmsCli)
    //fmt.Printf("======= Node [%s] finish delete hosts\n", node.Path)
    if isOk {
        //根节点不能删除
        if node.Value == jmsCli.OrgName {
            fmt.Printf("%s> *** root *** node [%s] keep stat here, forbidden delete...\n", jmsCli.OrgName, node.Path)
            isOk = true
        } else {
            fmt.Printf("%s> ===== Node [%s] start delete self\n", jmsCli.OrgName, node.Path)
            isOk = node.Delete(jmsCli)
        }
    }
    //fmt.Printf("=== Node [%s] finish delete\n", node.Path)
    return isOk
}

func (node *AssetsNode) Remove(jmsCli *JmsClient) bool {
    isSuccess := node.recursiveDelete(jmsCli)
    return isSuccess
}


// 获取子节点
func (node *AssetsNode) GetChild(jmsCli *JmsClient) (children []AssetsNode) {
    action := &JmsAction{
        Path: strings.Replace(request.JmsRoute["assets-nodes-children"], ":nid", node.Id, -1),
        Method: request.GET,
    }
    resp := jmsCli.JmsRequest(action)
    if resp.Success {
        err := convertToStruct(resp.Data, &children)
        if err != nil {
            fmt.Println("GetChild convertToStruct err:", err)
            children = nil
        }
    }
    return
}

//---- 搜索Assets
func CreateListAssetsNodesRequest() *AssetsNodesRequest {
    req := AssetsNodesRequest{
        Limit: 100000,      //直接获取所有,防止path找不到
        Offset: 0,
        Action: &JmsAction{
            Path: request.JmsRoute["assets-nodes"],
            Method: request.GET,
        },
    }
    return &req
}

func (jmsCli *JmsClient) ListAssetsNode(request *AssetsNodesRequest) (*AssetsNodesResponse, error) {
    query := buildHttpQuery(structs.Map(request))
    if query != "" {
        request.Action.Path = fmt.Sprintf("%s?%s", request.Action.Path, query)
    }
    assetsNodesResponse := AssetsNodesResponse{}
    resp := jmsCli.JmsRequest(request.Action)
    err := convertToStruct(resp.Data, &assetsNodesResponse)
    return &assetsNodesResponse, err
}

type JmsAnsibleResult struct {
    Cmd    string `json:"cmd"`
    Stderr string `json:"stderr"`
    Stdout string `json:"stdout"`
    Rc     int    `json:"rc"`
    Delta  string `json:"delta"`
    Err    string `json:"err"`
}

type JmsAnsibleResponse struct {
    Id           string                        `json:"id"`
    Hosts        []string                      `json:"hosts"`
    RunAs        string                        `json:"run_as"`
    Command      string                        `json:"command"`
    Result       map[string]JmsAnsibleResult   `json:"result"`
    LogUrl       string                        `json:"log_url"`
    IsFinished   bool                          `json:"is_finished"`
    DateCreated  string                        `json:"date_created"`
    DateFinished string                        `json:"date_finished"`
}

func (jmsCli *JmsClient) GetExecCommand(taskId string) (jmsAnsibleResponse *JmsAnsibleResponse, err error) {
    action := &JmsAction{
        Path: fmt.Sprintf("%s%s/", request.JmsRoute["command-exec"], taskId),
        Method: request.GET,
    }
    jmsAnsibleResponse = &JmsAnsibleResponse{}
    resp := jmsCli.JmsRequest(action)
    if resp.Success {
        err = convertToStruct(resp.Data, &jmsAnsibleResponse)
        if err != nil {
            fmt.Println("GetExecCommand convertToStruct err:", err)
        }
    } else {
        err = errors.New(resp.Err)
    }
    return
}

func (jmsCli *JmsClient) ExecCommand(hostIds []string, command string) (jmsAnsibleResponse *JmsAnsibleResponse, err error) {
    action := &JmsAction{
        Path: request.JmsRoute["command-exec"],
        Method: request.POST,
        Payload: map[string]interface{}{
            "hosts": hostIds,
            "run_as": config.G.JumpInfo.SystemId,
            "command": command,
        },
    }
    jmsAnsibleResponse = &JmsAnsibleResponse{}
    resp := jmsCli.JmsRequest(action)
    if resp.Success {
        err = convertToStruct(resp.Data, &jmsAnsibleResponse)
        if err != nil {
            fmt.Println("GetExecCommand convertToStruct err:", err)
        }
    } else {
        err = errors.New(resp.Err)
    }
    return
}

//----
func (jmsCli *JmsClient) JmsRequest(action *JmsAction) (response *JmsResponse){
    response = &JmsResponse{Success: false}
    action.Header = http.Header{}
    // 认证 + 固定租户
    action.Header.Set("Authorization", fmt.Sprintf("Token %s", jmsCli.PrivateToken))
    action.Header.Set("X-JMS-ORG", jmsCli.OrgId)
    if action.OrgId != "" {
        action.Header.Set("X-JMS-ORG", action.OrgId)
        action.OrgId = ""
    }


    jmsurl := jmsCli.EndPoint + action.Path

    r := request.SimpleHTTPClient{
        Url:           jmsurl,
        HeaderTimeout: time.Duration(60 * time.Second),
        Header:        action.Header,
    }

    resp, err := r.Do(nil, action)

    if err != nil {
        fmt.Printf("Request Jumpserver <%s> Err: %s\n", jmsurl, err.Error())
        response.Err = err.Error()
        return
    }

    defer resp.Body.Close()
    response.Code = resp.StatusCode
    if resp.StatusCode >= 400 {
        body, _ := ioutil.ReadAll(resp.Body)
        //fmt.Println(action.Payload)
        message := fmt.Sprintf("%s:%s Fail, Code: %d error: %s\n", action.Method, jmsurl, resp.StatusCode, string(body))
        fmt.Println(message)
        response.Err = message
        return
    }

    //Jump server DELETE 均不返回任何东西， 204成功
    if action.Method == request.DELETE && resp.StatusCode == 204 {
        response.Success = true
        return
    }

    err = json.NewDecoder(resp.Body).Decode(&response.Data)
    if err != nil {
        response.Err = fmt.Sprintf("%s: %s", "Response解析错误, 可能带有HTML内容", err.Error())
        //fmt.Println(action.Payload)
    }

    if response.Err != "" {
        fmt.Printf("[%d] %s:%s Faliure: %s\n", resp.StatusCode, action.Method, jmsurl, response.Err)
    } else {
        //fmt.Printf("Request N9e <%s> Success.\n", url)
        response.Success = true
    }
    return
}

func (jms *JmsAction) Body() string {
    body, _ := json.Marshal(jms.Payload)
    return string(body)
}

// DO会调用,实现了接口的一个结构体
func (jms *JmsAction) HTTPRequest(url string) *http.Request {
    r, _ := http.NewRequest(jms.Method, url, strings.NewReader(jms.Body()))
    /*
    r.Header.Add("date", time.Now().Format(time.RFC1123))
    err := SignRequest(JumpsCli.AccessKey, JumpsCli.SecretKey, r)
    if err != nil {
        fmt.Println("签名计算失败:", err)
    }
    */
    return r
}

//签名认证, 暂未通过
func SignRequest(APPKey string, APPSecret string, r *http.Request) error {
    privateKey := crypto.PrivateKey([]byte(APPSecret))
    algorithm := []httpsig.Algorithm{httpsig.HMAC_SHA256}
    //headersToSign := []string{"(request-target)", "accept", "date"}
    headersToSign := []string{"date"}
    signer, _, err := httpsig.NewSigner(algorithm, headersToSign, httpsig.Authorization)
    if err != nil {
        return err
    }
    return signer.SignRequest(privateKey, APPKey, r)
}