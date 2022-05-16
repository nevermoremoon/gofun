package config

const (
	Version = "v0.7.15" //cloud_essd PL0 Support => dexin machine register => ack eks scale => ReuqestFailCode => many tenant => k8s test
	AppMode = "debug"  //debug or release
	AppPort = "0.0.0.0:9999"
	AppName = "cloud-manager"
	// 签名超时时间
	AppSignExpiry = "120"

	// RSA Private File
	AppRsaPrivateFile = "rsa/private.pem"

	// 超时时间
	AppReadTimeout  = 120
	AppWriteTimeout = 120

	// 日志文件
	AppAccessLogName = "log/" + AppName + "-access.log"
	AppErrorLogName  = "log/" + AppName + "-error.log"
	AppGrpcLogName   = "log/" + AppName + "-grpc.log"

	// 系统告警邮箱信息
	SystemEmailUser = "aiming.cao@quanshi.com"
	SystemEmailPass = "" //密码或授权码
	SystemEmailHost = "smtp.quanshi.com"
	SystemEmailPort = 465

	// 告警接收人
	ErrorNotifyUser = "aiming.cao@quanshi.com"

	// 告警开关 1=开通 -1=关闭
	ErrorNotifyOpen = -1

	// Jaeger 配置信息
	JaegerHostPort = "127.0.0.1:6831"

	// Jaeger 配置开关 1=开通 -1=关闭
	JaegerOpen = 1
)

/* config模块全局变量 */
var (
	G *ConfYaml
	ApiAuthConfig = map[string] map[string]string {
		// 调用方
		"DEMO" : {
			"md5" : "IgkibX71IEf382PT",
			"aes" : "IgkibX71IEf382PT",
			"rsa" : "rsa/public.pem",
		},
	}
	DEFAULT = "default"
	MAJOR  = "quanshi"
)

type N9E struct {
	UserToken  string  `yaml:"userToken"`
	RdbToken   string  `yaml:"rdbToken"`
	Endpoint   string  `yaml:"endPoint"`
	AmsToken   string  `yaml:"amsToken"`
	PublicNid  string  `yaml:"publicNid"`
	DexinNid   string  `yaml:"dexinNid"`
	CdtsPodNid string  `yaml:"cdtsPodNid"`
	Tenant     string
	TenantList []*Tenant `yaml:"tenantList"`
	TenantMap  map[string]*Tenant
}

type Tenant struct {
	Name    string `yaml:"name"`
	Cname   string `yaml:"cname"`
	RootNid string `yaml:"rootNid"`
	Mapping string `yaml:"mapping"`
}

type Jumpserver struct {
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
	ExternalTenantList []*ExternalTenant `yaml:"externalTenant"`
	ExternalTenantMap map[string]*ExternalTenant
}

type ExternalTenant struct {
	OrgId        string `yaml:"orgId"`
	OrgName      string `yaml:"orgName"`
	AdminUser    string `yaml:"adminUser"`
	AdminId      string `yaml:"adminId"`
	SystemUser   string `yaml:"systemUser"`
	SystemId     string `yaml:"systemId"`
}

type Console struct {
	Endpoint      string `yaml:"endPoint"`
	FilebeatShell string  `yaml:"filebeatShell"`
}

type Kubernetes struct {
	TestConnectShell string `yaml:"testConnectShell"`
}

type Aws struct{
	Owner           string `yaml:"owner"`
	Name            string `name:"name"`
	RegionId        string `yaml:"regionId"`
	AccessKeyId     string `yaml:"accessKeyId"`
	AccessKeySecret string `yaml:"accessKeySecret"`
	TryCloneImageId string `yaml:"tryCloneImageId"`
	SetHostCommand  string `yaml:"setHostCommand"`
	S3              []S3Bucket `yaml:"s3"`
}

type S3Bucket struct {
	Bucket string `yaml:"bucket"`
	Object string `yaml:"object"`
}

type Aliyun struct {
	Owner           string `yaml:"owner"`
	Name            string `name:"name"`
	RegionId        string `yaml:"regionId"`
	AccessKeyId     string `yaml:"accessKeyId"`
	AccessKeySecret string `yaml:"accessKeySecret"`
	TryCloneImageId string `yaml:"tryCloneImageId"`
	SetHostCommand  string `yaml:"setHostCommand"`
}

type ConfYaml struct {
	N9eInfo       *N9E          `yaml:"n9e"`
	JumpInfo      *Jumpserver   `yaml:"jumpserver"`
	ConsoleInfo   *Console      `yaml:"console"`
	KubernetesInfo *Kubernetes  `yaml:"kubernetes"`
	AliyunInfo    []*Aliyun     `yaml:"aliyun"`
	AwsInfo       []*Aws        `yaml:"aws"`
	AliyunMap     map[string]*Aliyun
	AwsMap        map[string]*Aws
}

func(cf *ConfYaml) InitCloud() {
	cf.AliyunMap = make(map[string]*Aliyun)
	cf.AwsMap = make(map[string]*Aws)
	cf.JumpInfo.ExternalTenantMap = make(map[string]*ExternalTenant)
	cf.N9eInfo.TenantMap = make(map[string]*Tenant)

	for _, a := range cf.AliyunInfo {
		cf.AliyunMap[a.Owner] = a
	}
	for _, w := range cf.AwsInfo {
		cf.AwsMap[w.Owner] = w
	}

	for _, et := range cf.JumpInfo.ExternalTenantList {
		cf.JumpInfo.ExternalTenantMap[et.OrgName] = et
	}

	for _, t := range cf.N9eInfo.TenantList {
		cf.N9eInfo.TenantMap[t.Name] = t
	}
}