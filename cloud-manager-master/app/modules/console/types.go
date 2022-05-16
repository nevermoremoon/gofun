package console

import (
	"net/http"
)

var (
	ConsoleCli *ConsoleClient
)

type ConsoleClient struct {
	Endpoint  string  `yaml:"endPoint"`
	FilebeatShell string  `yaml:"filebeatShell"`
}

type ConsoleAction struct {
	Path          string
	Method        string
	Authenticate  string
	Header        http.Header
	Payload       interface{}
}

type ConsoleResponse struct {
	Err string       `json:"err"`
	Dat interface{}  `json:"dat"`
	Success bool
}
