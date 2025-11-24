// Package runtime LLM通信基础设施
package runtime

import (
	"net"
	"net/http"
	"time"

	"github.com/sohaha/zlsgo/zhttp"
)

var client = zhttp.New()

func init() {
	client.SetTransport(func(trans *http.Transport) {
		trans.DialContext = (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext
		trans.TLSHandshakeTimeout = 3 * time.Second
		trans.MaxIdleConns = 100
		trans.MaxIdleConnsPerHost = 10
		trans.IdleConnTimeout = 60 * time.Second
		trans.ExpectContinueTimeout = 1 * time.Second
	})
}

// SetClient 设置HTTP客户端
func SetClient(c *zhttp.Engine) {
	client = c
}

// GetClient 获取HTTP客户端
func GetClient() *zhttp.Engine {
	return client
}
