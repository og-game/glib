package limit

import (
	"fmt"
	"github.com/og-game/glib/stores/redisx"
	"github.com/og-game/glib/stores/redisx/config"
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	redisx.Must(config.Config{
		Addrs: []string{"192.168.110.149:6379"},
		Debug: true,
		Trace: true,

		Password: "redis123",
		DB:       0,
	})
}
func TestPeriodLimit_Take(t *testing.T) {
	testPeriodLimit(t)
}

func TestPeriodLimit_TakeWithAlign(t *testing.T) {
	testPeriodLimit(t, Align())
}

func TestPeriodLimit_RedisUnavailable(t *testing.T) {
	const (
		seconds = 5
		quota   = 5
	)

	l := NewPeriodLimit(seconds, quota, redisx.Engine, "periodlimit")
	redisx.Engine.Close()
	val, err := l.Take("first")
	assert.NotNil(t, err)
	assert.Equal(t, 0, val)
}

func testPeriodLimit(t *testing.T, opts ...PeriodOption) {
	//store := redistest.CreateRedis(t)
	//rdb, err := redis.NewRedis(redis.RedisConf{
	//	Host: "192.168.110.149:6379",
	//	Pass: "redis123",
	//	Type: "node",
	//})
	//assert.Nil(t, err)
	const (
		seconds = 5
		total   = 10
		quota   = 5
	)
	l := NewPeriodLimit(seconds, quota, redisx.Engine, "periodlimit", opts...)
	var allowed, hitQuota, overQuota int
	for i := 0; i < total; i++ {
		val, err := l.Take("first")
		if err != nil {
			t.Error(err)
		}
		fmt.Println(val)
		switch val {
		case Allowed:
			allowed++
		case HitQuota:
			hitQuota++
		case OverQuota:
			overQuota++
		default:
			t.Error("unknown status")
		}
	}

	assert.Equal(t, quota-1, allowed)
	assert.Equal(t, 1, hitQuota)
	assert.Equal(t, total-quota, overQuota)
}

func TestQuotaFull(t *testing.T) {

	l := NewPeriodLimit(1, 1, redisx.Engine, "periodlimit")
	val, err := l.Take("first")
	assert.Nil(t, err)
	assert.Equal(t, HitQuota, val)
}
