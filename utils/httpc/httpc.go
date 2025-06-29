package httpc

import (
	"context"
	"github.com/go-resty/resty/v2"
	"sync"
)

// 定义http 引擎
var engine *resty.Client
var once sync.Once

func Do(ctx context.Context) *resty.Request {
	once.Do(func() {
		engine = MustClient()
	})
	return engine.R().SetContext(ctx)
}

func New(ctx context.Context, fs ...func(cli *resty.Client)) *resty.Request {
	client := MustClient()
	for _, f := range fs {
		f(client)
	}
	return client.R().SetContext(ctx)
}

// MustClient new http client
func MustClient() *resty.Client {
	return resty.New()
}
