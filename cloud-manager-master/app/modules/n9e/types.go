package n9e

import (
	"cloud-manager/app/config"
	"fmt"
	"net/http"
)

type N9EClient struct {
	UserToken  string        `yaml:"userToken"`
	RdbToken   string        `yaml:"rdbToken"`
	Endpoint   string        `yaml:"endPoint"`
	AmsToken   string        `yaml:"amsToken"`
	PublicNid  string        `yaml:"publicNid"`
	DexinNid   string        `yaml:"dexinNid"`
	CdtsPodNid string        `yaml:"cdtsPodNid"`
	Tenant     string
	TenantList []*config.Tenant `yaml:"tenantList"`
	TenantMap  map[string]*config.Tenant
}

type N9EAction struct {
	Path          string
	Method        string
	Authenticate  string
	Header        http.Header
	Payload       interface{}
}

type N9EResponse struct {
	Err string       `json:"err"`
	Dat interface{}  `json:"dat"`
	Success bool
}

type AmsDat struct {
	List   []*N9EHost `json:"list"`
	Total  int        `json:"total"`
}

type N9ELeaf struct {
	List      []*N9EResource  `json:"list"`
	Total     int             `json:"total"`
	Reference map[string]int  /* 引用计数 */
}

type BindDat struct {
	RdbID int                   `json:"id"`
	Nodes []*N9ENode            `json:"nodes"`
	Name  string                `json:"name"`
	Dict  map[string]*N9ENode
}

type N9ENode struct {
	Nid      int       `json:"id"`
	PNid     int       `json:"pid"`
	Leaf     int       `json:"leaf"`
	Path     string    `json:"path"`
	Cate     string    `json:"cate"`
	Name     string    `json:"name"`
	Proxy    int       `json:"proxy"`
	NamePath string    `json:"namePath"`
	Hosts    []N9EHost `json:"hosts"`
}

type N9EPod struct {
	RdbID       int    `json:"id"`
	Ident       string `json:"ident"`
	Name        string `json:"name"`
	Tenant      string `json:"tenant"`
	Cate        string `json:"cate"`
	Note        string `json:"note"`
	UUID        string `json:"uuid"`
	Labels      string `json:"labels"`
	LabelMap    map[string]string
	IP          string
	HostIP      string
	Bind        *BindDat
}



type RdbHostDat *N9EHost



type N9EResource struct {
	RdbID    int      `json:"id"`
	Ident    string   `json:"ident"`
	Name     string   `json:"name"`
	Tenant   string   `json:"tenant"`
	Cate     string   `json:"cate"`
	Note     string   `json:"note"`
	Labels   string   `json:"labels"`
	UUID     string   `json:"uuid"`
}

//没有IP字段
type N9EHost struct {
	AmsID    int
	RdbID    int      `json:"id"`
	Ident    string   `json:"ident"`
	IP       string   `json:"ip"`
	Hostname string   `json:"name"`
	Tenant   string   `json:"tenant"`
	Cate     string   `json:"cate"`
	Note     string   `json:"note"`
	Clock    int      `json:"clock"`
	UUID     string   `json:"uuid"`
	Labels   string   `json:"labels"`
	Bind     *BindDat
}


type V1ContainersRegisterItem struct {
	UUID   string `json:"uuid"`
	Ident  string `json:"ident"`
	Name   string `json:"name"`
	Labels string `json:"labels"`
	Extend string `json:"extend"`
	Cate   string `json:"cate"`
	NID    int    `json:"nid"`
}

type HostRegisterForm struct {
	IP      string                 `json:"ip"`
	Ident   string                 `json:"ident"`
	Name    string                 `json:"name"`
}

func (h *HostRegisterForm) ToString() string {
	return fmt.Sprintf("%s::%s::%s", h.IP, h.Ident, h.Name)
}

func deepCopy(obj *config.N9E) *N9EClient {
	return &N9EClient{
		UserToken:  obj.UserToken,
		RdbToken:   obj.RdbToken,
		Endpoint:   obj.Endpoint,
		AmsToken:   obj.AmsToken,
		PublicNid:  obj.PublicNid,
		DexinNid:   obj.DexinNid,
		CdtsPodNid: obj.CdtsPodNid,
		Tenant:     obj.Tenant,
		TenantList: obj.TenantList,
		TenantMap:  obj.TenantMap,
	}
}