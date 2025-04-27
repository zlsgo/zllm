package inlay

import (
	"net/http"
	"time"

	"github.com/sohaha/zlsgo/zhttp"
)

var client = zhttp.New()

func init() {
	client.SetTransport(func(trans *http.Transport) {
		trans.TLSHandshakeTimeout = 3 * time.Second
	})
}

func SetClient(c *zhttp.Engine) {
	client = c
}

func GetClient() *zhttp.Engine {
	return client
}
