package route

import (
	"cloud-manager/app/controller/jaeger_conn"
	"cloud-manager/app/controller/product"
	"cloud-manager/app/controller/test"
	"cloud-manager/app/resource"
	"cloud-manager/app/route/middleware/exception"
	"cloud-manager/app/route/middleware/jaeger"
	"cloud-manager/app/route/middleware/logger"
	"cloud-manager/app/util/response"
	"github.com/gin-gonic/gin"
)

func SetupRouter(engine *gin.Engine) {

	//设置路由中间件
	engine.Use(logger.SetUp(), exception.SetUp(), jaeger.SetUp())

	//404
	engine.NoRoute(func(c *gin.Context) {
		utilGin := response.Gin{Ctx: c}
		utilGin.Response(404,"请求方法不存在", nil)
	})

	engine.GET("/ping", func(c *gin.Context) {
		utilGin := response.Gin{Ctx: c}
		utilGin.Response(1,"pong", nil)
	})

	// 测试链路追踪
	engine.GET("/jaeger_test", jaeger_conn.JaegerTest)

	//@todo 记录请求超时的路由

	ProductRouter := engine.Group("/product")
	{
		// 新增产品
		ProductRouter.POST("", product.Add)

		// 更新产品
		ProductRouter.PUT("/:id", product.Edit)

		// 删除产品
		ProductRouter.DELETE("/:id", product.Delete)

		// 获取产品详情
		ProductRouter.GET("/:id", product.Detail)
	}

	// 测试加密性能
	TestRouter := engine.Group("/test")
	{
		// 测试 MD5 组合 的性能
		TestRouter.GET("/md5", test.Md5Test)

		// 测试 AES 的性能
		TestRouter.GET("/aes", test.AesTest)

		// 测试 RSA 的性能
		TestRouter.GET("/rsa", test.RsaTest)
	}

	//resource
	Resource := engine.Group("/v1/resource")
	{
		Resource.GET("/vpc", resource.ListVpcs)
		Resource.GET("/vpc/:id/subnet", resource.ListSubnets)
		Resource.GET("/vpc/:id/security-group", resource.ListSecurityGroups)
		Resource.GET("/public-ipv4", resource.ListPublicIpv4s)
		Resource.PUT("/instance", resource.CreateInstance)
		Resource.GET("/instance", resource.ListInstances)
		Resource.GET("/disk", resource.ListDisks)
		Resource.DELETE("/instance", resource.AuthToTerminateInstances)
		Resource.GET("/instance/type", resource.ListInstanceTypes)
		Resource.GET("/instance/image", resource.ListImages)
		Resource.GET("/instance/type-family", resource.ListInstanceTypeFamilies)
		Resource.GET("/instance/keypair", resource.ListKeyPairs)
		Resource.GET("/group/resource", resource.ListGroups)
		Resource.PUT("/container", resource.ContainerRegister)
		Resource.DELETE("/container", resource.ContainerUnRegister)
		Resource.POST("/register", resource.NormalRegister)
		Resource.DELETE("/unregister", resource.NormalUnRegister)
	}

	Kubernetes := engine.Group("/v1/kubernetes")
	{
		Kubernetes.PUT("/nodegroup/scale", resource.NodeGroupScale)
		Kubernetes.PUT("/task/apiserver-test", resource.TestApiServerConnect)
	}

	Console := engine.Group("/v1/console")
	{
		Console.PUT("/log/filebeat", resource.OperateFilebeat)
		Console.GET("/n9e/hosts", resource.GetN9eHostsByNid)
	}

	SyncN9e := engine.Group("/v1/sync")
	{
		SyncN9e.GET("/n9e-tree", resource.SyncTree)
		SyncN9e.GET("/vendor-info", resource.SyncVendorInfo)
	}

	Backup := engine.Group("/v1/backup")
	{
		Backup.GET("/jump/hosts", resource.S3JumpBackup)
	}

	Info := engine.Group("/v1/info")
	{
		Info.GET("/n9e/hostname", resource.GetNodeInfo)
		Info.POST("/n9e/register", resource.HostRegister)
	}
}
