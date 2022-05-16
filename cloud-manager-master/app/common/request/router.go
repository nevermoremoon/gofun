package request

var N9ERoute map[string]string = map[string]string{
	"host-report": "/v1/ams-ce/hosts/register",                     // agent 注册主机
	"container-bind": "/v1/rdb/containers/bind",
	"container-unbind": "/v1/rdb/resources/unregister",
	"container-update": "/v1/rdb/container/sync",
	"leaf-search": "/v1/rdb/node/:nid/resources?limit=-1",                   // 节点下资源查询
    "host-tenant": "/api/ams-ce/hosts/tenant",                      // 管理员：设置租户
    "host-note-rdb": "/api/rdb/node/:nid/resources/note",           // 管理员：设置备注
	"host-note-ams": "/api/ams-ce/hosts/note",                      // 管理员：设置备注
    "host-bind": "/api/rdb/node/:nid/resources/bind",               // 挂载主机
    "host-unbind": "/api/rdb/node/:nid/resources/unbind",           // 解除挂载
	"host-search": "/api/rdb/resources/search?field=ident&batch=",  // 查询主机是否注册
	"host-init": "/api/ams-ce/hosts?query=",                        // 无租户资源
	"host-sync-binds": "/api/rdb/resources/bindings?ids=",
	"modify-label": "/api/rdb/node/:nid/resources/labels",          // 修改标签
	"host-register": "/api/ams-ce/hosts",                           // 主机注册
	"host-delete": "/api/ams-ce/hosts",                           // 下线删除 Delete
	"host-back": "/api/ams-ce/hosts/back",                          // 回收设备 Put
	"node": "/v1/rdb/node/:nid",                                    // 查指定资源
	"nodes": "/v1/rdb/nodes",
	"hosts": "/v1/rdb/node/:nid/resources?limit=-1",
	"job-ce": "/api/job-ce/tasks",
}

var ConsoleRoute map[string]string = map[string]string{
	"callback": "/api/v1/cloud/callback",
}

var JmsRoute map[string]string = map[string]string {
	"user-list": "/api/v1/users/users/",
	"assets-list":"/api/v1/assets/assets/",
	"assets-nodes":"/api/v1/assets/nodes/",
	"assets-assets": "/api/v1/assets/assets/",
	"assets-nodes-children": "/api/v1/assets/nodes/:nid/children/",
	"assets-nodes-add": "/api/v1/assets/nodes/:nid/assets/add/",
	"assets-nodes-remove": "/api/v1/assets/nodes/:nid/assets/remove/",
	"user-user": "/api/v1/users/users/",
	"user-group": "/api/v1/users/groups",
	"user-import":"/api/v1/users/users/invite/",
	"command-exec":"/api/v1/ops/command-executions/",
	"connect-test": "/api/v1/assets/assets/:id/tasks/",
}