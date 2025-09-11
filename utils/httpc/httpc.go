package httpc

import (
	"context"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cast"
	"github.com/zeromicro/go-zero/core/logx"

	"os"
	"sync"
)

// 定义http 引擎
var engine *resty.Client
var once sync.Once

func Do(ctx context.Context) *resty.Request {
	once.Do(func() {
		engine = MustClient()
	})

	debug := cast.ToBool(os.Getenv("HTTP_REQUEST_DEBUG"))
	trace := cast.ToBool(os.Getenv("HTTP_REQUEST_TRACE"))

	r := engine.R().
		SetContext(ctx).
		SetDebug(debug).
		SetLogger(CreateRestryLog(ctx))

	if trace {
		r = r.EnableTrace()
	}

	return r
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

type RestyLog struct {
	logx.Logger
}

func CreateRestryLog(ctx context.Context) *RestyLog {
	return &RestyLog{logx.WithContext(ctx)}
}

func (l RestyLog) Errorf(format string, v ...interface{}) {
	l.Logger.Errorf(format, v...)
}

func (l RestyLog) Warnf(format string, v ...interface{}) {
	l.Logger.Errorf(format, v...)
}

func (l RestyLog) Debugf(format string, v ...interface{}) {
	l.Logger.Infof(format, v...)
}
