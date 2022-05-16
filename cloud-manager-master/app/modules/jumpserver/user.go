package jumpserver

import (
	"cloud-manager/app/common/request"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fatih/structs"
)

type User struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	UserName string `json:"UserName"`
	Email    string `json:"email"`
	WeChat   string `json:"wechat"`
	UserGroups []string `json:"groups"`
	Phone      *string  `json:"phone"`
	MFALevel   int    `json:"mfa_level"`
	MFAEnabled bool   `json:"mfa_enabled"`
	MfaLevelDisplay  string `json:"mfa_level_display"`
	MfaForceEnabled  bool   `json:"mfa_force_enabled"`
	RoleDisplay      string `json:"role_display"`
	OrgRoleDisplay   string `json:"org_role_display"`
	TotalRoleDisplay string `json:"total_role_display"`
	Comment          string `json:"comment"`
	Source           string `json:"Source"`
	IsValid          bool   `json:"is_valid"`
	IsExpired        bool   `json:"is_expired"`
	IsActive         bool   `json:"is_active"`
	CreatedBy        string `json:"created_by"`
	IsFirstLogin     bool   `json:"is_first_log"`
	Role             string `json:"role"`
	GroupsDisplay    string `json:"groups_display"`
	CanUpdate        bool   `json:"can_update"`
	CanDelete        bool   `json:"can_delete"`
	OrgRoles         []string `json:"org_roles"`
}

type UserGroup struct {
    Id string  `json:"id"`
    Name string `json:"name"`
    Comment string `json:"comment"`
	CreatedBy string `json:"created_by"`
    Users []string `json:"users"`
	UsersAmount int `json:"users_amount"`
    OrgId string `json:"org_id"`
    OrgName string `json:"org_name"`
}


func (jmsCli *JmsClient) GetUserGroup() (userGroup []UserGroup ,err error) {
	action := &JmsAction{
		Path: request.JmsRoute["user-group"],
		Method: request.GET,
		OrgId: "00000000-0000-0000-0000-000000000002",
	}
	resp := jmsCli.JmsRequest(action)

	if resp.Success {
		err = convertToStruct(resp.Data, &userGroup)
	} else {
		err = errors.New(resp.Err)
	}
	return
}
func (jmsCli *JmsClient) GetUser() (user []User ,err error) {
	action := &JmsAction{
		Path: request.JmsRoute["user-user"],
		Method: request.GET,
		OrgId: "00000000-0000-0000-0000-000000000002",
	}
	resp := jmsCli.JmsRequest(action)

	if resp.Success {
		err = convertToStruct(resp.Data, &user)
	} else {
		err = errors.New(resp.Err)
	}
	return
}
type UserRequest struct {
	Action     *JmsAction
	Name       string
	UserName   string
	Email      string
	Id         string
	Search     string     //模糊匹配
	Order      string
	Spm        string
	Limit      int
	Offset     int
}

type UserResponse struct {
	Results  []User         `json:"results"`
	Previous string         `json:"previous"`
	Next     string         `json:"next"`
	Count    int            `json:"count"`
}

func CreateUserRequest() *UserRequest {
	req := UserRequest{
		Limit: 100000,
		Offset: 0,
		Action: &JmsAction{
			Path: request.JmsRoute["user-user"],
			Method: request.GET,
		},
	}
	return &req
}

func (jmsCli *JmsClient) NewUser(username string)(user *User, err error) {
	req := CreateUserRequest()
	req.UserName = username
	resp, err := jmsCli.ListUser(req)
	b, _ := json.Marshal(resp)
	fmt.Println(string(b))
	if err == nil {
		for _, u := range resp.Results {
			if u.UserName == username {
				user = &u
				break
			}
		}
	}
	return user, err
}

func (jmsCli *JmsClient) ListUser(request *UserRequest) (*UserResponse, error){
	query := buildHttpQuery(structs.Map(request))
	if query != "" {
		request.Action.Path = fmt.Sprintf("%s?%s", request.Action.Path, query)
	}
	fmt.Println(request.Action.Path)
	userResponse := UserResponse{}
	resp := jmsCli.JmsRequest(request.Action)
	err := convertToStruct(resp.Data, &userResponse)
	return &userResponse, err
}

type UserGroupRequest struct {
	Action     *JmsAction
	Name       string
	Ids        string
	Search     string     //模糊匹配
	Order      string
	Spm        string
	Limit      int
	Offset     int
}

type UserGroupResponse struct {
	Results  []UserGroup    `json:"results"`
	Previous string         `json:"previous"`
	Next     string         `json:"next"`
	Count    int            `json:"count"`
}

func CreateUserGroupRequest() *UserGroupRequest {
	req := UserGroupRequest{
		Limit: 100000,
		Offset: 0,
		Action: &JmsAction{
			Path: request.JmsRoute["user-group"],
			Method: request.GET,
		},
	}
	return &req
}

func (jmsCli *JmsClient) ListUserGroup(request *UserGroupRequest) (*UserGroupResponse, error){
	query := buildHttpQuery(structs.Map(request))
	if query != "" {
		request.Action.Path = fmt.Sprintf("%s?%s", request.Action.Path, query)
	}
	fmt.Println(request.Action.Path)
	userGroupResponse := UserGroupResponse{}
	resp := jmsCli.JmsRequest(request.Action)
	err := convertToStruct(resp.Data, &userGroupResponse)
	return &userGroupResponse, err
}

func (jmsCli *JmsClient) NewUserGroup(name string, id string)(userGroup *UserGroup, err error) {
	req := CreateUserGroupRequest()
	if name == "" && id == "" {
		err = errors.New("NewUserGroup is at least need one field, name or id")
		return
	}
	if name != "" {
		req.Name = name
	}
	if id != "" {
		req.Ids = id
	}
	resp, err := jmsCli.ListUserGroup(req)
	b, _ := json.Marshal(resp)
	fmt.Println(string(b))
	if err == nil {
		for _, ug := range resp.Results {
			if ug.Name == name {
				userGroup = &ug
				break
			}
		}
	}
	return userGroup, err
}

type UserRegisterParam struct {
	Name      string
	UserName  string
	Email     string
	WeChat    string
	Phone     string
	MFALevel  int
	Comment   string
	Source    string
	IsActive  bool
	groups    []string
	Role      string
	OrgRoles  []string
}


func (jmsCli *JmsClient) AddUser(param UserRegisterParam) (err error) {
	paramMap := map[string]interface{} {
		"name": param.Name,
		"username": param.UserName,
		"email": param.Email,
		"wechat": param.WeChat,
		"phone": param.Phone,
		"mfa_level": param.MFALevel,
		"comment": param.Comment,
		"source": param.Source,
		"is_active": param.IsActive,
		"groups": param.groups,
		"role": param.Role,
		"org_roles": param.OrgRoles,
	}
	action := JmsAction{
		Path: request.JmsRoute["user-user"],
		Method: request.POST,
		Payload: paramMap,
	}
	resp := jmsCli.JmsRequest(&action)
	if !resp.Success {
		err = errors.New(resp.Err)
	}
	return err
}

func (jmsCli *JmsClient) LoadUser(user string, role string) (err error){
	action := JmsAction{
		Path: request.JmsRoute["user-import"],
		Method: request.POST,
		Payload: []map[string]string {
			{"user": user, "role": role},
		},
	}
	resp := jmsCli.JmsRequest(&action)
	if !resp.Success {
		err = errors.New(resp.Err)
	}
	return err
}

func (jmsCli *JmsClient) SyncUser() {
	defaultUserGroupMap := map[string]*UserGroup{}
	defaultUserGroups, err := jmsCli.GetUserGroup()
	if err != nil {
		fmt.Println("Get all group Err:", err)
		return
	}
	for _, g := range defaultUserGroups {
		defaultUserGroupMap[g.Id] = &g
	}

	defaultUsers, err := jmsCli.GetUser()
	if err != nil {
		fmt.Println("Get all user Err:", err)
		return
	}
	for _, u := range defaultUsers {
		newUser, _ :=jmsCli.NewUser(u.UserName)
		if newUser != nil {
			fmt.Printf("User:%s Already exists, ignore...\n", newUser.Name)
			continue
		} else {
			/*
			var newGroupIds []string
			for _, og := range u.UserGroups {
				newGroup, err := NewUserGroup("", og)
				if err == nil && newGroup != nil {
					newGroupIds = append(newGroupIds, newGroup.Id)
				}
			}
			*/


			if u.UserName == "admin" {
				fmt.Println("user [admin], ignore...")
				continue
			}
/*
			param := UserRegisterParam{
				Name:     u.Name,
				UserName: u.UserName,
				Email:    u.Email,
				WeChat:   u.WeChat,
				MFALevel: u.MFALevel,
				Comment:  u.Comment,
				Source:   u.Source,
				IsActive: u.IsActive,
				groups:   newGroupIds,
				Role:     u.Role,
				OrgRoles: u.OrgRoles,
			}
			if u.Phone != nil {
				param.Phone = *u.Phone
			}
            err = AddUser(param)
*/
			err = jmsCli.LoadUser(u.Id, u.Role)
			if err != nil {
				fmt.Printf("User[%s] import err: %s\n", u.UserName, err.Error())
			} else {
				fmt.Printf("User[%s] import success.\n", u.UserName)
			}
		}
	}
}
