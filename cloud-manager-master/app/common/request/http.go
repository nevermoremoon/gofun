package request

import (
    "context"
    "errors"
    "net/http"
    "time"
)

var (
        ErrNoAvailableEndpoints = errors.New("httpClient: No available endpoints.")
        ErrAuthenticationFailed = errors.New("httpClient: Authentication failed.")
        ErrBadRequest           = errors.New("httpClient: Bad Request.")
        ErrServerInternal       = errors.New("httpClient: Server internal error.")
)

const (
    GET    = http.MethodGet
    POST   = http.MethodPost
    PUT    = http.MethodPut
    DELETE = http.MethodDelete
)


type httpClient interface {
    Do(context.Context, httpAction) *http.Response
}

type httpAction interface {
    HTTPRequest(url string) *http.Request
}

type SimpleHTTPClient struct {
    Url           string
    HeaderTimeout time.Duration
    Header        http.Header
}

func (s *SimpleHTTPClient) Do(ctx context.Context, act httpAction) (*http.Response, error) {
    req := act.HTTPRequest(s.Url)
    /* 默认设置 */
    req.Header.Set("Content-Type", "application/json")
    /* 自定义设置 */
    for k, v := range s.Header {
        req.Header.Set(k, v[0])
    }

    cli := http.Client{
        Timeout: s.HeaderTimeout,
    }
    resp, err := cli.Do(req)

    if resp == nil && err == nil {
        err = errors.New("Empty reply from server")
    }

    return resp, err
}


