package httpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"time"

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
		engine = MustClient().
			SetRetryCount(1).
			AddRetryCondition(func(_ *resty.Response, err error) bool {
				return errors.Is(err, syscall.ECONNRESET)
			}).
			AddRetryHook(logRetryOutcome)
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
	client := resty.New()
	transport := client.GetClient().Transport.(*http.Transport)
	transport.IdleConnTimeout = 60 * time.Second
	return client
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

// logRetryOutcome records errors selected by the retry conditions. Resty calls
// retry hooks even when no retries remain, so this does not imply another send.
func logRetryOutcome(resp *resty.Response, err error) {
	ctx := context.Background()
	fields := []logx.LogField{
		logx.Field("error_type", fmt.Sprintf("%T", err)),
		logx.Field("is_reset", errors.Is(err, syscall.ECONNRESET)),
		logx.Field("retry_condition_matched", true),
	}
	if resp != nil && resp.Request != nil {
		req := resp.Request
		ctx = req.Context()
		fields = append(fields, logx.Field("attempt", req.Attempt),
			logx.Field("callback_request_id", req.Header.Get("X-Request-ID")),
			logx.Field("method", req.Method))
		if target, parseErr := url.Parse(req.URL); parseErr == nil {
			fields = append(fields, logx.Field("host", target.Host), logx.Field("path", target.EscapedPath()))
		}
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		fields = append(fields, logx.Field("net_op", opErr.Op), logx.Field("net_network", opErr.Net))
		if opErr.Source != nil {
			fields = append(fields, logx.Field("net_source", opErr.Source.String()))
		}
		if opErr.Addr != nil {
			fields = append(fields, logx.Field("net_addr", opErr.Addr.String()))
		}
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		err = urlErr.Err
	}
	if err != nil {
		fields = append(fields, logx.Field("error", err.Error()))
	}
	logx.WithContext(ctx).Errorw("http retry condition matched", fields...)
}
