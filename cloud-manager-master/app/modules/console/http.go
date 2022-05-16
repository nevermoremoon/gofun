package console

import (
	"cloud-manager/app/common/request"
	"cloud-manager/app/config"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

func NewConosleClient(obj *config.Console) *ConsoleClient {
	ConsoleCli = (*ConsoleClient)(obj)
	return ConsoleCli
}

//异步调用
func (c *ConsoleClient) Callback(instance interface{}) {
	action := ConsoleAction{
		Path: request.ConsoleRoute["callback"],
		Method: request.POST,
		Payload: instance,
	}
	fmt.Println("callback----------")
	b, _ := json.Marshal(instance)
	fmt.Println(string(b))
	go func(ac *ConsoleAction) {
		resp := consoleRequest(ac)
		fmt.Println("callback-resp=", resp.Dat)
		fmt.Println("------------------")
	}(&action)
}

// 5s超时
func consoleRequest(action *ConsoleAction) (response *ConsoleResponse){
	response = &ConsoleResponse{Success: false}
	action.Header = http.Header{}
	action.Header.Set("Content-Type", "application/json")
	var message string

	url := ConsoleCli.Endpoint + action.Path

	r := request.SimpleHTTPClient{
		Url:           url,
		HeaderTimeout: time.Duration(5 * time.Second),
		Header:        action.Header,
	}
	resp, err := r.Do(nil, action)

	if err != nil {
		fmt.Printf("Request Console <%s> Err: %s\n", url, err.Error())
		response.Err = err.Error()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		//fmt.Println(action.Payload)
		message = fmt.Sprintf("%s:%s Fail, Code: %d\nMessage: %s\n", action.Method, url, resp.StatusCode, string(body))
		fmt.Println(message)
		response.Err= message
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

func (ca *ConsoleAction) Body() string {
	body, _ := json.Marshal(ca.Payload)
	return string(body)
}

// DO会调用,实现了接口的一个结构体
func (ca *ConsoleAction) HTTPRequest(url string) *http.Request {
	r, _ := http.NewRequest(ca.Method, url, strings.NewReader(ca.Body()))
	return r
}