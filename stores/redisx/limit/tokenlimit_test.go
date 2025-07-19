package limit

import (
	"context"
	"github.com/og-game/glib/stores/redisx"
	"github.com/og-game/glib/stores/redisx/config"
	"github.com/zeromicro/go-zero/core/limit"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/logx"
)

func init() {
	logx.Disable()
}

func TestTokenLimit_WithCtx(t *testing.T) {

	const (
		total = 100
		rate  = 5
		burst = 10
	)
	l := NewTokenLimiter(rate, burst, redisx.Engine, "tokenlimit")
	defer redisx.Engine.Close()

	ctx, cancel := context.WithCancel(context.Background())
	ok := l.AllowCtx(ctx)
	assert.True(t, ok)

	cancel()
	for i := 0; i < total; i++ {
		ok := l.AllowCtx(ctx)
		assert.False(t, ok)
		assert.False(t, l.monitorStarted)
	}
}

func TestTokenLimit_Rescue(t *testing.T) {
	limit.Align()
	const (
		total = 100
		rate  = 5
		burst = 10
	)
	l := NewTokenLimiter(rate, burst, redisx.Engine, "tokenlimit")
	redisx.Engine.Close()

	var allowed int
	for i := 0; i < total; i++ {
		time.Sleep(time.Second / time.Duration(total))
		if i == total>>1 {
			//assert.Nil(t, s.Restart())
			redisx.Must(config.Config{
				Addrs: []string{"192.168.110.149:6379"},
				Debug: true,
				Trace: true,

				Password: "redis123",
				DB:       1,
			})
		}
		if l.Allow() {
			allowed++
		}

		// make sure start monitor more than once doesn't matter
		l.startMonitor()
	}

	assert.True(t, allowed >= burst+rate)
}

func TestTokenLimit_Take(t *testing.T) {
	//store := redistest.CreateRedis(t)

	const (
		total = 100
		rate  = 5
		burst = 10
	)
	l := NewTokenLimiter(rate, burst, redisx.Engine, "tokenlimit")
	var allowed int
	for i := 0; i < total; i++ {
		time.Sleep(time.Second / time.Duration(total))
		if l.Allow() {
			allowed++
		}
	}

	assert.True(t, allowed >= burst+rate)
}

func TestTokenLimit_TakeBurst(t *testing.T) {
	//store := redistest.CreateRedis(t)

	const (
		total = 100
		rate  = 5
		burst = 10
	)
	l := NewTokenLimiter(rate, burst, redisx.Engine, "tokenlimit")
	var allowed int
	for i := 0; i < total; i++ {
		if l.Allow() {
			allowed++
		}
	}

	assert.True(t, allowed >= burst)
}
