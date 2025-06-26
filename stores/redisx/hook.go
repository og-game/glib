package redisx

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logc"
	"net"
)

type DebugHook struct {
}

func (d DebugHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return next(ctx, network, addr)
	}
}
func (d DebugHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		logc.Debugf(ctx, "redis cmd: %s", cmd.String())
		return next(ctx, cmd)
	}
}

func (d DebugHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		logc.Debugf(ctx, "redis cmd: %s", d.cmdToString(cmds))
		return next(ctx, cmds)
	}
}

// 组装cmd
func (DebugHook) cmdToString(cmds []redis.Cmder) []string {
	var cmdsStr []string
	for _, cmd := range cmds {
		cmdsStr = append(cmdsStr, cmd.String())
	}
	return cmdsStr
}
