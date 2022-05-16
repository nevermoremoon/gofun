package jumpserver

import (
	"cloud-manager/app/config"
	"net/http"
)

type JmsClient struct {
	EndPoint     string `yaml:"endPoint"`
	AccessKey    string `yaml:"accessKey"`
	SecretKey    string `yaml:"secretKey"`
	OrgId        string `yaml:"orgId"`
	OrgName      string `yaml:"orgName"`
	PrivateToken string `yaml:"privateToken"`
	AdminUser    string `yaml:"adminUser"`
	AdminId      string `yaml:"adminId"`
	SystemUser   string `yaml:"systemUser"`
	SystemId     string `yaml:"systemId"`
	ExternalTenantList []*config.ExternalTenant `yaml:"externalTenant"`
	ExternalTenantMap map[string]*config.ExternalTenant
}

type JmsAction struct {
	Path          string
	Method        string
	OrgId         string
	Header        http.Header
	Payload       interface{}
}


type JmsResponse struct {
	Code     int
	Err      string
	Data     interface{}
	Success  bool
}

func deepCopy(obj *config.Jumpserver) *JmsClient {
	return &JmsClient{
		EndPoint:           obj.EndPoint,
		AccessKey:          obj.AccessKey,
		SecretKey:          obj.SecretKey,
		OrgId:              obj.OrgId,
		OrgName:            obj.OrgName,
		PrivateToken:       obj.PrivateToken,
		AdminUser:          obj.AdminUser,
		AdminId:            obj.AdminId,
		SystemUser:         obj.SystemUser,
		SystemId:           obj.SystemId,
		ExternalTenantList: obj.ExternalTenantList,
		ExternalTenantMap:  obj.ExternalTenantMap,
	}
}