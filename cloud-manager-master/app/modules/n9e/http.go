package n9e

import (
    "cloud-manager/app/common/request"
    "cloud-manager/app/config"
    "cloud-manager/app/modules/jumpserver"
    "encoding/json"
    "errors"
    "fmt"
    "github.com/wxnacy/wgo/arrays"
    "io/ioutil"
    "net/http"
    "strconv"
    "strings"
    "time"
)

func NewN9EClient(obj *config.N9E, tenant string) (n9eCli *N9EClient) {
    n9eCli = deepCopy(obj)
    n9eCli.Tenant = config.MAJOR
    if tenant != "" && tenant != config.MAJOR {
        if t, ok := n9eCli.TenantMap[tenant]; ok {
            n9eCli.Tenant = t.Name
        }
    }
    return
}


//下线机器不在使用hostname， 只用IP
func (n9eCli *N9EClient) OfflineJms(ip, hostname string) error {
    fmt.Println("Start offline Jms...")
    defer func() {
        panicErr := recover()
        if panicErr != nil {
            fmt.Println("捕获到异常=>", panicErr)
        }
    }()

    jmsCli := jumpserver.NewJmsClient(config.G.JumpInfo, n9eCli.Tenant)
    hosts, err := jmsCli.NewAssetsHostByIP(ip)
    if err == nil {
        for i := 0; i < len(hosts); i++ {
            _ = hosts[i].OffLine(jmsCli)
            fmt.Printf("host:%s offline from n9e\n", hosts[i].Ip)
        }
    }
    fmt.Println("Jms Offline finish.")
    return err
}


func (n9eCli *N9EClient) RegisterJms(ip, hostname, vendor string, nidList []int) (string, error) {
    fmt.Printf("[%s] jms register...\n", ip)
    defer func() {
        panicErr := recover()
        if panicErr != nil {
            fmt.Println("捕获到异常=>", panicErr)
        }
    }()
    //var jmsNodeIds  []string
    //var jmsNodePath []string
    var err error
    var errMsg string
    var virtualHostname string
    jmsCli := jumpserver.NewJmsClient(config.G.JumpInfo, n9eCli.Tenant)
    for _, nid := range nidList {
        nidStr := strconv.Itoa(nid)
        n9eNode, err := n9eCli.NewNode(nidStr)
        if err == nil && n9eNode != nil {
            n9eNode.SetNamePath(n9eCli)
            jmsNode, err := jmsCli.NewAssetsNode(n9eNode.NamePath)
            if err != nil {
                errMsg = fmt.Sprintf("Jms NewAssetsNode() fail, jmsNode=<%v>, err: %v", jmsNode, err)
                fmt.Println(errMsg)
                continue
            }
            //不存在，需创建全路径
            if jmsNode == nil {
                jmsNode = jmsCli.MakeJmsFullPath(n9eNode.NamePath)
            }
            // 仍然为空则放弃
            if jmsNode != nil {
                //jmsNodeIds = append(jmsNodeIds, jmsNode.Id)
                //jmsNodePath = append(jmsNodePath, jmsNode.Path)
                //开始注册,重新计算名称,防止挂载在多处,hostname不具有代表性
                hostPrefix := n9eNode.GetHostPrefix()
                if hostPrefix != "" {
                    ipSlice := strings.Split(ip, ".")
                    ipLen := len(ipSlice)
                    virtualHostname = fmt.Sprintf("%s-%s-%s", hostPrefix, ipSlice[ipLen-2], ipSlice[ipLen-1])
                }

                var host *jumpserver.AssetsHost
                host, err = jmsCli.AddHost(jumpserver.RegisterParam{
                    Ip:       ip,
                    HostName: virtualHostname,
                    Nodes:    []string{jmsNode.Id},
                    Vendor:   vendor,
                    Comment:  jmsNode.Path,
                })
                if err == nil {
                    b, _ := json.Marshal(host)
                    fmt.Println(">>Jms注册成功:", string(b))
                } else {
                    if strings.Contains(err.Error(), "字段必须唯一") {
                        fmt.Println(">>Jms Hosts注册成功: Already exists...")
                        err = nil
                    } else {
                        errMsg = fmt.Sprintf("Jms AddHost [%s] fail, %s", ip, err.Error())
                        fmt.Println(errMsg)
                    }
                }
            }
        } else {
            errMsg = fmt.Sprintf("N9e Newnode() fail, err:%v\n", err)
        }
    }
    /*
    if jmsNodeIds != nil && len(jmsNodeIds) > 0 {
        var comment string
        if jmsNodePath != nil {
            comment = strings.Join(jmsNodePath, ", ")
        }

        //未计算出主机名
        if hostname == "" {
            hostname = ip
        }
        var host *jumpserver.AssetsHost
        host, err = jmsCli.AddHost(jumpserver.RegisterParam{
            Ip:       ip,
            HostName: hostname,
            Nodes:    jmsNodeIds,
            Vendor:   vendor,
            Comment:  comment,
        })
        if err == nil {
            b, _ := json.Marshal(host)
            fmt.Println(">>Jms注册成功:", string(b))
        } else {
            if strings.Contains(err.Error(), "字段必须唯一") {
                fmt.Println(">>Jms Hosts注册成功: Already exists...")
                err = nil
            } else {
                errMsg = fmt.Sprintf("Jms AddHost [%s] fail, %s", ip, err.Error())
                fmt.Println(errMsg)
            }
        }
    } else {
        err = fmt.Errorf("RegisterJms Err: %s", errMsg)
    }
    */
    fmt.Println("Finish register jumpserver...")
    return virtualHostname, err
}

func (n9eCli *N9EClient) RegisterHost(host HostRegisterForm) (n9eHost *N9EHost, err error) {
    //format[Ip::Ident::Name]
    hostLine := []string{host.ToString()}
    action := N9EAction{
        Path:         request.N9ERoute["host-register"],
        Method:       request.POST,
        Payload:      hostLine,
        Authenticate: "user",
    }
    resp := n9eCli.Request(&action)
    if !resp.Success {
        /* 早已存在 */
        alreadyStr := fmt.Sprintf("Duplicate entry '%s' for key 'ip'", host.IP)
        if strings.Contains(resp.Err, alreadyStr) {
            resp.Success = true
        }
    }
    if resp.Success {
        n9eHost = n9eCli.NewHost(host.Ident)
    } else {
        err = errors.New(resp.Err)
    }
    return
}


func (host *N9EHost) Json() string {
    byteSlice, _ := json.Marshal(host)
    return string(byteSlice)
}


func (leaf *N9ELeaf) Json() string {
    byteSlice, _ := json.Marshal(leaf)
    return string(byteSlice)
}

func (leaf *N9ELeaf) HostNeedUnmount(host string) bool {
    if leaf.Reference[host] == 1 {
        return true
    }
    return false
}


//是否随同Pod完成挂载
func (host *N9EHost) IsMountK8S(nid int) bool {
    if host.Bind != nil {
        _, ok := host.Bind.Dict[strconv.Itoa(nid)]
        return ok
    }
    return false
}


//资源绑定关系，单独拆解出来
func SyncBinddins(rid int, ident string, n9e *N9EClient) (bind *BindDat) {
    if rid < 0 {
        return
    }
    bind = nil
    action := N9EAction {
        Path: request.N9ERoute["host-sync-binds"]+strconv.Itoa(rid),
        Method: request.GET,
        Authenticate: "user",
    }
    resp := n9e.Request(&action)
    if resp.Success {
        byteSlice, _ := json.Marshal(resp.Dat)
        bindDat := []BindDat{}
        if err := json.Unmarshal(byteSlice, &bindDat); err != nil {
            fmt.Printf("Init BinDat <%s> Fail: %s\n", ident, err.Error())
            return
        }

        /* 每次只查1个资源的绑定关系 */
        if len(bindDat) > 0 {
            bind = &(bindDat[0])
            bind.Dict = map[string]*N9ENode{}
            for _, node := range bind.Nodes {
                bind.Dict[strconv.Itoa(node.Nid)] = node
            }
        }
    }
    return
}

/* 搜索游离资源，不存在直接初始化资源 */
func (n9eCli *N9EClient) InitialHost(ident string) *N9EHost {
    host := &N9EHost{AmsID: -1, RdbID: -1}
    //n9e.Host = nil  /* 置空 */
    action := N9EAction{
        Path: request.N9ERoute["host-init"]+ident,
        Method: request.GET,
        Authenticate: "user",
    }
    resp := n9eCli.Request(&action)
    if resp.Success {
        /* 精确匹配ident, 符合Agent上报规范 */
        byteSlice, _ := json.Marshal(resp.Dat)
        amsDat := &AmsDat{}
        if err := json.Unmarshal(byteSlice, amsDat); err != nil {
            fmt.Printf("Init AmsDat <%s> Fail: %s\n", ident, err.Error())
        } else {
            if amsDat.Total > 0 {
                for _, v := range amsDat.List {
                    if v.Ident == ident {
                        host = v
                        host.AmsID = v.RdbID
                        host.RdbID = -1
                        break
                    }
                }
            }
        }
    }
    return host
}


func(host *N9EHost)BackDevice(n9eCli *N9EClient) (*N9EResponse, error) {
    fmt.Printf("N9e: 开始回收设备<%s>\n", host.IP)
    var err error
    var resp *N9EResponse
    if host.AmsID < 0 {
        err = errors.New(fmt.Sprintf("N9e: Host id <%d> is valid, host not found.", host.AmsID))
    } else {
        action := N9EAction{
            Path: request.N9ERoute["host-back"],
            Method: request.PUT,
            Authenticate: "user",
            Payload: map[string]interface{}{
                "ids": []int{host.AmsID},
            },
        }
        resp = n9eCli.Request(&action)
        if !resp.Success {
            err = errors.New(resp.Err)
        }
    }
    return resp, err
}

//type N9eNodeSet []N9ENode

func (n9eCli *N9EClient) SearchNode(nid string) (*N9ENode, error) {
    // 必须传nid，获取所有
    var node N9ENode
    var err error
    action := N9EAction{
        Path:         strings.Replace(request.N9ERoute["node"], ":nid", nid, -1),
        Method:       request.GET,
        Authenticate: "rdb",
    }

    resp := n9eCli.Request(&action)
    if resp.Success {
        err = convertToStruct(resp.Dat, &node)
    } else {
        err = fmt.Errorf(resp.Err)
    }
    return &node, err
}

//初始化Node实例
func (n9eCli *N9EClient) NewNode(nid string) (*N9ENode, error) {
    node, err := n9eCli.SearchNode(nid)
    return node, err
}


type NodeHostsResponse struct {
    List []N9EHost `json:"list"`
}

func (node *N9ENode) GetHosts(n9eCli *N9EClient) (err error) {
    action := N9EAction{
        Path:         strings.Replace(request.N9ERoute["leaf-search"], ":nid", strconv.Itoa(node.Nid), -1),
        Method:       request.GET,
        Authenticate: "rdb",
    }
    nodeHostsResponse := NodeHostsResponse{}
    resp := n9eCli.Request(&action)

    if resp.Success {
        err = convertToStruct(resp.Dat, &nodeHostsResponse)
        if err != nil {
            fmt.Printf("Node<%s> GetHosts function convertToStruct err: %s\n", node.Path, err)
            return
        }
        node.Hosts = nodeHostsResponse.List
    } else {
        fmt.Printf("Node<%s> GetHosts err: %s\n", node.Path, resp.Err)
        err = errors.New(resp.Err)
    }
    return err
}

func (host *N9EHost) GetVirtualHostname(node *N9ENode) (hostname string) {
    prefix := node.GetHostPrefix()
    if prefix == "" {
        fmt.Printf("[%s/%s] get hostprefix empty, set host.IP\n", node.Path, host.Ident)
        hostname = host.Ident
        return
    }
    ipSlice := strings.Split(host.Ident, ".")
    ipLen := len(ipSlice)
    hostname = fmt.Sprintf("%s-%s-%s", prefix, ipSlice[ipLen-2], ipSlice[ipLen-1])
    return
}

//得到Name路径
func (node *N9ENode) SetNamePath(n9eCli *N9EClient) {
    var pathSlice []string
    visitNode := node
    for {
        if visitNode == nil {
            fmt.Println("Recursive node path, find nil, stop.")
            return
        }
        if visitNode.PNid == 0 {
            pathSlice = append([]string{visitNode.Name}, pathSlice...)
            break
        }
        pathSlice = append([]string{visitNode.Name}, pathSlice...)
        visitNode, _ = n9eCli.NewNode(strconv.Itoa(visitNode.PNid))
    }
    node.NamePath = fmt.Sprintf("/%s", strings.Join(pathSlice, "/"))
    //fmt.Println(node.NamePath)
}

func (node *N9ENode) SetNamePathByMap(nodeMap map[int]*N9ENode) {
    var pathSlice []string
    visitNode := node
    for {
        if visitNode == nil {
            fmt.Println("Recursive node path, find nil, stop.")
            return
        }
        if visitNode.PNid == 0 {
            pathSlice = append([]string{visitNode.Name}, pathSlice...)
            break
        }
        pathSlice = append([]string{visitNode.Name}, pathSlice...)
        visitNode, _ = nodeMap[visitNode.PNid]
    }
    node.NamePath = fmt.Sprintf("/%s", strings.Join(pathSlice, "/"))
    //fmt.Println(node.NamePath)
}

func (node *N9ENode) GetHostPrefix() (prefix string) {
    if !node.IsLeaf() {
        fmt.Printf("[%s]为非叶子节点，不支持计算hostname\n", node.Path)
        return
    }
    if needIgnore(node) {
        fmt.Printf("[%s]为排除目录，不支持计算hostname\n", node.Path)
        return
    }
    pathSlice := strings.Split(node.Path, ".")
    pathLen := len(pathSlice)
    if pathLen >= 2 {
        prefix = fmt.Sprintf("%s-%s", pathSlice[pathLen-1], pathSlice[pathLen-2])
        //容器节点
        if strings.HasPrefix(node.Path, "quanshi.basecomp.k8s") {
            prefix = fmt.Sprintf("%s-%s", pathSlice[pathLen-2], pathSlice[pathLen-1])
        }
    }
    return
}

func (node *N9ENode) GetHostPrefixWithError() (prefix string, err error) {
    if !node.IsLeaf() {
        err = fmt.Errorf("nid:%d:%s为非叶子节点，不支持计算hostname", node.Nid, node.Path)
        return
    }
    if needIgnore(node) {
        err = fmt.Errorf("nid:%d:%s为排除目录，不支持计算hostname", node.Nid, node.Path)
        return
    }
    pathSlice := strings.Split(node.Path, ".")
    pathLen := len(pathSlice)
    if pathLen >= 2 {
        prefix = fmt.Sprintf("%s-%s", pathSlice[pathLen-1], pathSlice[pathLen-2])
        //容器节点
        if strings.HasPrefix(node.Path, "quanshi.basecomp.k8s") {
            prefix = fmt.Sprintf("%s-%s", pathSlice[pathLen-2], pathSlice[pathLen-1])
        }
    }
    return
}

func (n9eCli *N9EClient) GetAllNodes() (n9eNodes []*N9ENode, err error) {
    action := N9EAction{
        Path: request.N9ERoute["nodes"],
        Method: request.GET,
        Authenticate: "rdb",
    }
    resp := n9eCli.Request(&action)
    if resp.Success {
        err = convertToStruct(resp.Dat, &n9eNodes)
        if err != nil {
            fmt.Println("GetAllNodes function convertToStruct err:", err)
            return
        }
    } else {
        err = fmt.Errorf(resp.Err)
    }
    return
}

func (n9eCli *N9EClient )GetNodePods(nid int) (n9ePods []*N9EPod, err error) {
    action := N9EAction{
        Path: strings.Replace(request.N9ERoute["hosts"], ":nid", strconv.Itoa(nid), -1),
        Method: request.GET,
        Authenticate: "user",
    }
    var  podList struct {
        List []*N9EPod `json:"list"`
    }
    resp := n9eCli.Request(&action)
    if resp.Success {
        err = convertToStruct(resp.Dat, &podList)
        if err != nil {
            fmt.Println("GetTreeHosts function convertToStruct err:", err)
            return
        }
        n9ePods = podList.List
    } else {
        err = fmt.Errorf(resp.Err)
    }
    return
}

func (n9eCli *N9EClient)GetNodeHosts(nid int) (n9eHosts []*N9EHost, err error) {
    action := N9EAction{
        Path: strings.Replace(request.N9ERoute["hosts"], ":nid", strconv.Itoa(nid), -1),
        Method: request.GET,
        Authenticate: "rdb",
    }
    var  hostList struct {
        List []*N9EHost `json:"list"`
    }
    resp := n9eCli.Request(&action)
    if resp.Success {
        err = convertToStruct(resp.Dat, &hostList)
        if err != nil {
            fmt.Println("GetTreeHosts function convertToStruct err:", err)
            return
        }
        n9eHosts = hostList.List
    } else {
        err = fmt.Errorf(resp.Err)
    }
    return
}

type TreeAdd struct {
    NodeList  []*N9ENode
    NodeMap   map[int]*N9ENode
    ChildMap  map[int][]int
}

func needIgnore(node *N9ENode) bool {
    if strings.HasPrefix(node.Path,"inner") {
        return true
    }
    if node.Cate == "containercluster" && strings.Contains(node.Path, "appcenter") {
        return true
    }
    if strings.HasPrefix(node.Path,"quanshi.basecomp.transmitting") {
        return true
    }
    if strings.HasPrefix(node.Path, "quanshi.appcenter.kubernetes") {
        return true
    }
    //不再处理机器中心
    if strings.HasPrefix(node.Path, "quanshi.machinecenter") {
        return true
    }
    return false
}

func (node *N9ENode) IsLeaf() bool {
    if node.Leaf == 1 {
        return true
    }
    return false
}

//=======
type TreeAdd2 struct {
    N9eNidMap       map[int]*N9ENode
    N9ePathMap      map[int]string     //nid对应的hostname
    N9eChildMap     map[int][]int
    JmsPathMap      map[string]*jumpserver.AssetsNode
    JmsHostMap      map[string][]*jumpserver.AssetsHost
}


func (tree *TreeAdd2) InitialN9eMap(n9eCli *N9EClient) (err error) {
    var n9eNodes []*N9ENode
    n9eNodes, err = n9eCli.GetAllNodes()
    //生成n9e Map
    if err == nil {
        tree.N9eNidMap = make(map[int]*N9ENode)
        tree.N9ePathMap = make(map[int]string)
        tree.N9eChildMap = make(map[int][]int)
        for _, node := range n9eNodes {
            if needIgnore(node) {
                continue
            }
            //if rootId > 0 {
            //    if node.Nid == rootId  {
            //        root = node
            //    }
            //} else {
            //    if node.PNid == 0 {
            //        root = node
            //    }
            //}
            tree.N9eNidMap[node.Nid] = node
            tree.N9eChildMap[node.PNid] = append(tree.N9eChildMap[node.PNid], node.Nid)
        }
        for _, node := range tree.N9eNidMap {
            node.SetNamePathByMap(tree.N9eNidMap)
            if node.NamePath != "" {
                tree.N9ePathMap[node.Nid] = node.NamePath
            }
        }
    }
    return
}

func (tree *TreeAdd2) InitialJmsMap(jms *jumpserver.JmsClient) (err error) {
    tree.JmsPathMap = make(map[string]*jumpserver.AssetsNode)
    tree.JmsHostMap = make(map[string][]*jumpserver.AssetsHost)
    //jms开始递归建树
    jmsNodes := jms.GetAllNodes()
    if len(jmsNodes) <= 0 {
        err = fmt.Errorf("SyncAdd InitialJmsMap GetAllNodes empty.")
        return
    }

    for _, jmsNode := range jmsNodes {
        tree.JmsPathMap[jmsNode.Path] = jmsNode
    }
    //获取叶子节点下的机器
    jmsHosts := jms.GetAllHosts()
    for _, jmsHost := range jmsHosts {
        for _, path := range jmsHost.NodesPath {
            tree.JmsHostMap[path] = append(tree.JmsHostMap[path], jmsHost)
        }
    }
    return
}

func (n9eCli *N9EClient) RecursiveToJumpServer2(root *N9ENode, tree *TreeAdd2, rootPath string, jmsCli *jumpserver.JmsClient) {
    if root == nil {
        return
    }
    jmsNode, ok := tree.JmsPathMap[rootPath]
    if !ok {
        //找不到当前节点，则没法创建子孩子,所以必须存在
        fmt.Printf("[%s] can't find in jms, 404 ignore...\n", rootPath)
        return
    }

    if !root.IsLeaf() {
        //非叶子节点，可能有子孩子,有则遍历
        if _, ok := tree.N9eChildMap[root.Nid]; ok {
            for _, childId := range tree.N9eChildMap[root.Nid] {
                // 能找到孩子对象
                if n9eChild, ok := tree.N9eNidMap[childId]; ok {
                    jmsChild, ok := tree.JmsPathMap[n9eChild.NamePath]
                    //不存在，创建
                    if !ok {
                        //fmt.Printf("=== Sync node [%s]\n", rootPath)
                        fmt.Printf("====== make create [%s]\n", n9eChild.NamePath)
                        jmsChild = jmsNode.CreateChildren(n9eChild.Name, jmsCli)
                    }
                    // jms那边创建路径成功，开始递归处理n9e该child
                    if jmsChild != nil {
                        tree.JmsPathMap[n9eChild.NamePath] = jmsChild
                        n9eCli.RecursiveToJumpServer2(n9eChild, tree, n9eChild.NamePath, jmsCli)
                    }
                }
            }
        }
    } else {
        //资产（主机）注册
        err := root.GetHosts(n9eCli)
        if err == nil {
            var jmsNames []string
            jmsHosts := tree.JmsHostMap[root.NamePath]
            for _, jmsHost := range jmsHosts {
                jmsNames = append(jmsNames, jmsHost.Hostname)
            }

            for _, n9eHost := range root.Hosts {
                // 实例化jms的host
                hostname := n9eHost.GetVirtualHostname(root)
                if arrays.ContainsString(jmsNames, hostname) < 0 {
                    var jmsHost *jumpserver.AssetsHost
                    fmt.Printf("***** Host %s:<%s> sync to jms %s\n", hostname, n9eHost.Ident, rootPath)
                    jmsHost, err = jmsCli.AddHost(jumpserver.RegisterParam{
                       Ip:       n9eHost.Ident,
                       HostName: hostname,
                       Nodes:    []string{jmsNode.Id},
                       Comment:  jmsNode.Path,
                    })
                    if err != nil {
                       tree.JmsHostMap[hostname] = append(tree.JmsHostMap[hostname], jmsHost)
                       fmt.Printf("register host %s to %s err:%s\n", hostname, root.NamePath, err.Error())
                    }
                } else {
                    //fmt.Printf("===Host %s:<%s> already have [%s], ignore...\n", hostname, n9eHost.Ident, jmsNode.Path)
                    continue
                }
            }
        }
    }
}

func (n9eCli *N9EClient)SyncAdd(root *N9ENode) (err error) {
    //var path string
    tree := TreeAdd2{}
    jmsCli := jumpserver.NewJmsClient(config.G.JumpInfo, n9eCli.GetOrgName())
    err = tree.InitialN9eMap(n9eCli)
    if err == nil {
        if root.NamePath == "" {
            err = fmt.Errorf("root node [%s] mapping to jmspath is empty", root.Path)
        } else {
            //初始化Jms结构
            err = tree.InitialJmsMap(jmsCli)
        }
    }

    if err != nil {
        fmt.Printf("%s> sync add, err: %s\n", n9eCli.GetOrgName(), err.Error())
        return
    }
    // 开始递归建树
    fmt.Printf("%s> start add sync from root [%s]...\n", n9eCli.GetOrgName(), root.NamePath)
    n9eCli.RecursiveToJumpServer2(root, &tree, root.NamePath, jmsCli)

    return err
}

type TreeClean struct {
    N9eNodeList     []*N9ENode
    N9eNidMap       map[int]*N9ENode
    N9ePathMap      map[string]*N9ENode
    JmsPathMap      map[string]*jumpserver.AssetsNode
    JmsKeyMap       map[string]*jumpserver.AssetsNode
    JmsChildMap     map[string][]string
    JmsHostMap      map[string][]*jumpserver.AssetsHost
}

func (tree *TreeClean) InitialN9eMap(n9eCli *N9EClient) (err error) {
    var n9eNodes []*N9ENode
    n9eNodes, err = n9eCli.GetAllNodes()
    //生成n9e Map
    if err == nil {
        tree.N9eNidMap = make(map[int]*N9ENode)
        tree.N9ePathMap = make(map[string]*N9ENode)
        for _, node := range n9eNodes {
            nodeT := node
            if needIgnore(nodeT) {
                continue
            }
            tree.N9eNidMap[nodeT.Nid] = nodeT
        }

        for _, node := range tree.N9eNidMap {
            nodeT := node
            nodeT.SetNamePathByMap(tree.N9eNidMap)
            if nodeT.NamePath != "" {
                tree.N9ePathMap[nodeT.NamePath] = nodeT
            }
        }
    } else {
        fmt.Println("Get n9e nodes err", err)
    }
    return err
}

func (tree *TreeClean) InitialJmsMap(rootPath string, jmsCli *jumpserver.JmsClient) (jmsRoot *jumpserver.AssetsNode, err error) {
    tree.JmsPathMap = make(map[string]*jumpserver.AssetsNode)
    tree.JmsKeyMap = make(map[string]*jumpserver.AssetsNode)
    tree.JmsChildMap = make(map[string][]string)
    tree.JmsHostMap = make(map[string][]*jumpserver.AssetsHost)

    //var jmsRoot *jumpserver.AssetsNode
    if rootPath != "" {
        jmsRoot, err = jmsCli.NewAssetsNode(rootPath)
        if err == nil && jmsRoot == nil {
            err = fmt.Errorf("Root node:%s not found.", rootPath)
        }
        if err != nil {
            return
        }
    }
    //jms开始递归建树
    jmsNodes := jmsCli.GetAllNodes()
    for _, jmsNode := range jmsNodes {
        tree.JmsPathMap[jmsNode.Path] = jmsNode
        tree.JmsKeyMap[jmsNode.Key] = jmsNode
        pKey := jmsNode.ParentKey()
        if pKey != "" {
            tree.JmsChildMap[pKey] = append(tree.JmsChildMap[pKey], jmsNode.Key)
        }
        //没指定根节点选择顶级节点
        if pKey == "" && jmsRoot == nil {
            jmsRoot  = jmsNode
            rootPath = jmsNode.Path
        }
    }
    //获取叶子节点下的机器
    jmsHosts := jmsCli.GetAllHosts()
    for _, jmsHost := range jmsHosts {
        for _, path := range jmsHost.NodesPath {
            tree.JmsHostMap[path] = append(tree.JmsHostMap[path], jmsHost)
        }
    }
    return
}

func (n9eCli *N9EClient)SyncClean(rootPath string) (err error) {
    tree := TreeClean{}
    jmsCli := jumpserver.NewJmsClient(config.G.JumpInfo, n9eCli.GetOrgName())
    var jmsRoot *jumpserver.AssetsNode
    err = tree.InitialN9eMap(n9eCli)
    if err == nil {
        jmsRoot, err = tree.InitialJmsMap(rootPath, jmsCli)
    }

    if err != nil || jmsRoot == nil {
        return
    }

    fmt.Printf("%s> start clean sync root [%s] ...\n", jmsCli.OrgName, jmsRoot.Path)
    //执行根节点
    n9eCli.RecursiveToClean(jmsRoot, &tree, jmsRoot.Path, jmsCli)

    fmt.Printf("===EOF Clear End...\n")
    return err
}


func (n9eCli *N9EClient) GetOrgName() (orgName string) {
    tenant := n9eCli.Tenant
    if t, ok := n9eCli.TenantMap[tenant]; ok {
        if t.Name == config.MAJOR {
            orgName = config.DEFAULT
        } else {
            orgName = t.Cname
        }
    }
    return
}

func (n9eCli *N9EClient) RecursiveToClean(rootNode *jumpserver.AssetsNode, tree *TreeClean, rootPath string, jmsCli *jumpserver.JmsClient) {
    if rootNode == nil {
        return
    }
    //fmt.Println("recursive==", rootNode.Path, rootPath, rootNode.Key)
    n9eNode, ok := tree.N9ePathMap[rootPath]
    if ok {
        children, ok := tree.JmsChildMap[rootNode.Key]
        if ok && len(children) > 0 {
            for _, child := range children {
                if childNode, ok := tree.JmsKeyMap[child]; ok {
                    n9eCli.RecursiveToClean(childNode, tree, childNode.Path, jmsCli)
                }
            }
        }
        //是叶子节点，开始扫描机器,发现被删除机器
        if n9eNode.IsLeaf() {
            var err error
            var n9eIps []string
            var hostPrefix string

            hostPrefix = n9eNode.GetHostPrefix()
            err = n9eNode.GetHosts(n9eCli)
            //获取n9e节点下资产失败，不能进行对比
            if err != nil {
                return
            }
            for _, host := range n9eNode.Hosts {
                //保证是物理机
                n9eIps = append(n9eIps, host.Ident)
            }
            //err = rootNode.GetHosts()
            if jmsNodeHosts, ok := tree.JmsHostMap[rootPath]; ok {
                for _, host := range jmsNodeHosts {
                    if arrays.ContainsString(n9eIps, host.Ip) > -1 {
                        //清除掉IP匹配，但是hostname前缀不匹配的历史遗留（改了n9e ident但是没改name）造成的
                        if hostPrefix == "" || strings.HasPrefix(host.Hostname, hostPrefix) {
                            continue
                        }
                    }
                    fmt.Printf("%s> Host [%s] <%s:%s> need delete.\n", jmsCli.OrgName, rootNode.Path, host.Hostname, host.Ip)
                    _ = host.OffLine(jmsCli)
                }
            }
        }
    } else {
        //递归删除该Node
        fmt.Printf("%s> Delete path:<%s>\n", jmsCli.OrgName, rootPath)
        rootNode.Remove(jmsCli)
    }
}

// 删除设备
func (host *N9EHost)DeleteDevice(n9e *N9EClient) (*N9EResponse, error) {
    fmt.Printf("N9e: 开始下线设备<%s>\n", host.IP)
    var err error
    var resp *N9EResponse
    if host.AmsID < 0 {
        err = errors.New(fmt.Sprintf("N9e: Host id <%d> is valid, host not found.", host.AmsID))
    } else {
        action := N9EAction{
            Path: request.N9ERoute["host-delete"],
            Method: request.DELETE,
            Authenticate: "user",
            Payload: map[string]interface{}{
                "ids": []int{host.AmsID},
            },
        }
        resp = n9e.Request(&action)
        if !resp.Success {
            err = errors.New(resp.Err)
        }
    }
    return resp, err
}


//下线
func (host *N9EHost)Offline(n9eCli *N9EClient) error {
    var err error
    _, err = host.BackDevice(n9eCli)
    if err == nil {
        _, err = host.DeleteDevice(n9eCli)
    }
    return err
}

func (host *N9EHost) SetRdb(n9eCli *N9EClient) {
    if host.AmsID > 0 {
        action := N9EAction{
            Path: request.N9ERoute["host-search"] + host.Ident,
            Method: request.GET,
            Authenticate: "user",
        }
        resp := n9eCli.Request(&action)
        if resp.Success {
            byteSlice, _ := json.Marshal(resp.Dat)
            rdbDat := []RdbHostDat{}
            if err := json.Unmarshal(byteSlice, &rdbDat); err != nil {
                fmt.Printf("Init RdbDat <%s> Fail: %s\n", host.Ident, err.Error())
            } else {
                if len(rdbDat) > 0 {
                    host.RdbID = rdbDat[0].RdbID
                    host.UUID = rdbDat[0].UUID
                    host.Tenant = rdbDat[0].Tenant
                    host.Cate = rdbDat[0].Cate
                    host.Clock = rdbDat[0].Clock
                    host.Note = rdbDat[0].Note
                    host.Bind = SyncBinddins(host.RdbID, host.Ident, n9eCli)
                }
            }
        }
    }
}

//刷新host信息
func (host *N9EHost) Refresh(n9eCli *N9EClient) {
    host.SetRdb(n9eCli)
}

//初始化host实例
func (n9eCli *N9EClient) NewHost(ident string) *N9EHost {
    host := n9eCli.InitialHost(ident)
    host.SetRdb(n9eCli)
    if host.AmsID < 0 {
        host = nil
    }
    return host
}

//绑定host至某个Leaf
func (host *N9EHost) HostBind(nid int, n9eCli *N9EClient) *N9EResponse {
    action := N9EAction{
        Path: strings.Replace(request.N9ERoute["host-bind"], ":nid", strconv.Itoa(nid), -1),
        Method: request.POST,
        Authenticate: "user",
        Payload: map[string]interface{} {
            "field": "ident",
            "items": []string {host.Ident},
        },
    }
    resp := n9eCli.Request(&action)
    return resp
}

//解除绑定host从某个Leaf， 有个接口可以处理所有
func (host *N9EHost) HostUnbind(nid int, n9eCli *N9EClient) bool {
    action := N9EAction{
        Path: strings.Replace(request.N9ERoute["host-unbind"], ":nid", strconv.Itoa(nid), -1),
        Method: request.POST,
        Authenticate: "user",
        Payload: map[string]interface{} {
            "ids": []int {host.RdbID},
        },
    }
    resp := n9eCli.Request(&action)
    return resp.Success
}

//更新Pod标签
func (pod *N9EPod) UpdateLabel(nid int, n9eCli *N9EClient) bool {
    action := N9EAction{
        Path: strings.Replace(request.N9ERoute["modify-label"], ":nid", strconv.Itoa(nid), -1),
        Method: request.PUT,
        Authenticate: "user",
        Payload: map[string]interface{} {
            "labels": pod.Labels,
            "ids": []int {pod.RdbID},
        },
    }
    resp := n9eCli.Request(&action)
    return resp.Success
}

//更新host标签
func (host *N9EHost) UpdateLabel(nid int, n9eCli *N9EClient) bool {
    action := N9EAction{
        Path: strings.Replace(request.N9ERoute["modify-label"], ":nid", strconv.Itoa(nid), -1),
        Method: request.PUT,
        Authenticate: "user",
        Payload: map[string]interface{} {
            "labels": host.Labels,
            "ids": []int {host.RdbID},
        },
    }
    resp := n9eCli.Request(&action)
    return resp.Success
}

/* 管理员：设置主机租户 */
func (host *N9EHost) SetHostTenant(tenant string, refresh bool, n9eCli *N9EClient) *N9EResponse {
    action := N9EAction{
        Path: request.N9ERoute["host-tenant"],
        Method: request.PUT,
        Authenticate: "user",
        Payload: map[string]interface{} {
            "tenant": tenant,
            "ids": []int {host.AmsID},
        },
    }
    resp := n9eCli.Request(&action)
    /* 设置完成后，获取RDB-ID */
    if refresh && resp.Success {
        host.Refresh(n9eCli)
        if host.RdbID < 0 {
            resp.Success = false
            resp.Err = "获取RDB信息失败，请联系管理员"
        }
    }
    return resp
}

/* 管理员：设置主机备注 */
func (host *N9EHost) SetHostAmsNote(note string, n9eCli *N9EClient) bool {
    action := N9EAction{
        Path: request.N9ERoute["host-note-ams"],
        Method: request.PUT,
        Authenticate: "user",
        Payload: map[string]interface{} {
            "note": note,
            "ids": []int {host.AmsID},
        },
    }
    resp := n9eCli.Request(&action)
    return resp.Success
}

/* 管理员：设置主机备注 */
func (host *N9EHost) SetHostRdbNote(nid int, n9eCli *N9EClient) bool {
    action := N9EAction{
        Path: strings.Replace(request.N9ERoute["host-note-rdb"], ":nid", strconv.Itoa(nid), -1),
        Method: request.PUT,
        Authenticate: "user",
        Payload: map[string]interface{} {
            "labels": host.Note,
            "ids": []int {host.RdbID},
        },
    }
    resp := n9eCli.Request(&action)
    return resp.Success
}

//TODO: 回收设备， 下线删除

// Leaf下resource labels重新调整
func (rs *N9EResource) LabelDict() map[string]string {
    labelDict := make(map[string]string)
    for _, v := range strings.Split(rs.Labels, ",") {
        label := strings.Split(v, "=")
        switch len(label) {
        case 1:
            labelDict[label[0]] = ""
        case 2:
            labelDict[label[0]] = label[1]
        default:
            ;
        }
    }
    return labelDict
}

//Leaf下挂载资源扫描
func (n9eCli *N9EClient) NewLeaf(nid int) (leaf *N9ELeaf) {
    leaf = nil
    action := N9EAction{
        Path: strings.Replace(request.N9ERoute["leaf-search"], ":nid", strconv.Itoa(nid), -1),
        Method: request.GET,
        Authenticate: "rdb",
    }
    resp := n9eCli.Request(&action)
    if resp.Success {
        /* 精确匹配ident, 符合Agent上报规范 */
        leaf = &N9ELeaf{Reference: make(map[string]int)}
        byteSlice, _ := json.Marshal(resp.Dat)
        if err := json.Unmarshal(byteSlice, leaf); err != nil {
            fmt.Printf("Init Leaf <%d> Fail: %s\n", nid, err.Error())
        } else {
            if leaf.Total > 0 {
                for _, rs := range leaf.List {
                    if rs.Cate == "container" {
                        if node, ok := rs.LabelDict()["node_ip"]; ok {
                            leaf.Reference[node] += 1
                        }
                    }
                }
            }
        }
    }
    return
}


type JobCe struct {
    Title     string   `json:"title"`
    Account   string   `json:"account"`
    Batch     int      `json:"batch"`
    Tolerance int      `json:"tolerance"`
    Timeout   int      `json:"timeout"`
    Pause     string   `json:"pause"`
    Hosts     []string `json:"hosts"`
    Script    string   `json:"script"`
    Args      string   `json:"args"`
    Tags      string   `json:"tags"`
    Action    string   `json:"action"`
}

type RunJobTaskRequest struct {
    Action *N9EAction
    Param  *JobCe
}

func CreateRunJobTaskRequest() *RunJobTaskRequest {
    req := RunJobTaskRequest{
        Action: &N9EAction{
            Path: request.N9ERoute["job-ce"],
            Method: request.POST,
            Authenticate: "user",
        },
        Param: &JobCe {
            Title:     "",
            Account:   "root",
            Batch:     0,
            Tolerance: 0,
            Timeout:   300,
            Pause:     "",
            Hosts:     []string{},
            Script:    "",
            Args:      "",
            Tags:      "",
            Action:    "start",
        },
    }
    req.Action.Payload = req.Param
    return &req
}

func (n9eCli *N9EClient) RunJobTask(title, args, script string, hosts []string) (taskId int, err error) {
    scriptPrefix := "#!/bin/bash\n# e.g.\nexport PATH=/usr/local/bin:/bin:/usr/bin:/usr/local/sbin:/usr/sbin:/sbin:~/bin\nss -tln\n"
    req := CreateRunJobTaskRequest()
    req.Param.Title = title
    req.Param.Script = fmt.Sprintf("%s\n%s", scriptPrefix, script)
    req.Param.Args = args
    req.Param.Hosts = hosts
    resp := n9eCli.Request(req.Action)
    if resp.Success {
        taskId = int(resp.Dat.(float64))
    } else {
        err = fmt.Errorf("%s", resp.Err)
    }
    return
}

type JobHost struct {
    Id     int    `json:"id"`
    Host   string `json:"host"`
    Status string `json:"status"`
    Stdout string `json:"stdout"`
    Stderr string `json:"stderr"`
}

type JobTask struct {
    Action string    `json:"action"`
    Hosts  []JobHost `json:"hosts"`
    Meta   TaskMeta  `json:"meta"`
}

type TaskMeta struct {
    Title     string   `json:"title"`
    Account   string   `json:"account"`
    Batch     int      `json:"batch"`
    Tolerance int      `json:"tolerance"`
    Timeout   int      `json:"timeout"`
    Pause     string   `json:"pause"`
    Script    string   `json:"script"`
    Args      string   `json:"args"`
    Tags      string   `json:"tags"`
    Action    string   `json:"start"`
    Id        int      `json:"id"`
    Creator   string   `json:"creator"`
    Created   string   `json:"created"`
    Done      bool     `json:"done"`
}

func (n9eCli *N9EClient) GetJobTaskDetail(taskId int) (jobTask JobTask, err error){
    action := N9EAction{
        Path: strings.Replace(request.N9ERoute["job-ce"], "tasks", fmt.Sprintf("task/%d",taskId), -1),
        Method: request.GET,
        Authenticate: "user",
    }
    resp := n9eCli.Request(&action)
    jobTask = JobTask{}
    if resp.Success {
        err = convertToStruct(resp.Dat, &jobTask)
    } else {
        err = fmt.Errorf("%s", resp.Err)
    }
    return
}

//func (n9e *N9EClient) request(action *N9EAction) (*http.Response, error) {
func (n9eCli *N9EClient) Request(action *N9EAction) (response *N9EResponse){
    //action.Header = http.Header{"x-user-token": []string{"56cb511dfcb3a82e7efd5ff5a0f641ed"}}
    //action.Header = http.Header{"X-Srv-Token": []string{"ams-builtin-token"}}
    //action.Header = http.Header{"X-Srv-Token": []string{"rdb-builtin-token"}}
    response = &N9EResponse{Success: false}
    action.Header = http.Header{}

    switch action.Authenticate {
        case "rdb":
            //action.Header.Set("X-Srv-Token", "rdb-builtin-token")
            action.Header.Set("X-Srv-Token", n9eCli.RdbToken)
        case "ams":
            //action.Header.Set("X-srv-Token", "ams-builtin-token")
            action.Header.Set("X-srv-Token", n9eCli.AmsToken)
        case "user":
            //action.Header.Set("x-user-token", "7d251e7712a642f63bf7a56d3d1ad4f5")
            action.Header.Set("x-user-token", n9eCli.UserToken)
        default:
            fmt.Printf("Unknow Authenticate type: <%s>\n", action.Authenticate)
    }

    url := n9eCli.Endpoint + action.Path
    r := request.SimpleHTTPClient{
        Url:           url,
        HeaderTimeout: time.Duration(30 * time.Second),
        Header:        action.Header,
    }

    resp, err := r.Do(nil, action)
    if err != nil {
        fmt.Printf("Request N9e <%s> Err: %s\n", url, err.Error())
        response.Err = err.Error()
        return
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        body, _ := ioutil.ReadAll(resp.Body)
        fmt.Println(action.Payload)
        fmt.Printf("%s:%s Fail, Code: %d\nMessage: %s\n", action.Method, url, resp.StatusCode, string(body))
        return
    }

    /* 会重置Success 字段  */
    err = json.NewDecoder(resp.Body).Decode(response)
    if err != nil {
        //fmt.Printf("Request N9e <%s> Read Body Err: %s", url, err.Error())
        response.Err = err.Error()
    }

    if response.Err != "" {
        fmt.Printf("%s:%s Faliure: %s\n", action.Method, url, response.Err)
        response.Success = false
    } else {
        //fmt.Printf("Request N9e <%s> Success.\n", url)
        response.Success = true
    }
    return
}

func (na *N9EAction) Body() string {
    body, _ := json.Marshal(na.Payload)
    return string(body)
}

// DO会调用,实现了接口的一个结构体
func (na *N9EAction) HTTPRequest(url string) *http.Request {
    r, _ := http.NewRequest(na.Method, url, strings.NewReader(na.Body()))
    return r
}