package utils

import "github.com/sohaha/zlsgo/zhttp"

var client = zhttp.New()

func SetClient(c *zhttp.Engine) {
	client = c
}

func GetClient() *zhttp.Engine {
	return client
}
