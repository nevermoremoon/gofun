package response

import "github.com/gin-gonic/gin"

type Gin struct {
	Ctx *gin.Context
}

type Response struct {
	Code     int         `json:"code"`
	Message  string      `json:"msg"`
	Data     interface{} `json:"data"`
}

type N9eResponse struct {
	Code    int         `json:"code"`
	Err     string      `json:"err"`
	Dat     interface{} `json:"dat"`
}
func (g *Gin)Response(code int, msg string, data interface{}) {
	g.Ctx.JSON(code, Response{
		Code    : code,
		Message : msg,
		Data    : data,
	})
	return
}

func (g *Gin)N9eResponse(code int, err string, dat interface{}) {
	g.Ctx.JSON(code, N9eResponse{
		Code : code,
		Err  : err,
		Dat  : dat,
	})
	return
}
