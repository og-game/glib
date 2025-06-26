package redisx

import (
	"context"
	"fmt"
	"new/glib-new/stores/redisx/config"
	"testing"
)

func TestRedis(t *testing.T) {
	Must(config.Config{
		Addrs:    []string{"10.0.2.73:6379"},
		Debug:    true,
		Trace:    true,
		Password: "QiYhFcRAJW",
	})
	r, err := Engine.Get(context.Background(), "test").Result()
	fmt.Println(err)
	fmt.Println(r)
}
