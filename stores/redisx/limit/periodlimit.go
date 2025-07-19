package limit

import (
	"context"
	_ "embed"
	"errors"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"

	"strconv"
	"time"
)

const (
	// Unknown means not initialized state.
	Unknown = iota
	// Allowed means allowed state.
	Allowed
	// HitQuota means this request exactly hit the quota.
	HitQuota
	// OverQuota means passed the quota.
	OverQuota

	internalOverQuota = 0
	internalAllowed   = 1
	internalHitQuota  = 2
)

var (
	// ErrUnknownCode is an error that represents unknown status code.
	ErrUnknownCode        = errors.New("unknown status code")
	ErrMaxRetriesExceeded = errors.New("period limit: max retries exceeded")
	ErrQuotaExhausted     = errors.New("period limit: quota exhausted")

	//go:embed periodscript.lua
	periodLuaScript string
	periodScript    = redis.NewScript(periodLuaScript)
)

type (
	// PeriodOption defines the method to customize a PeriodLimit.
	PeriodOption func(l *PeriodLimit)

	// A PeriodLimit is used to limit requests during a period of time.
	PeriodLimit struct {
		period     int
		quota      int
		limitStore redis.UniversalClient
		keyPrefix  string
		align      bool
	}
)

// NewPeriodLimit returns a PeriodLimit with given parameters.
func NewPeriodLimit(period, quota int, limitStore redis.UniversalClient, keyPrefix string,
	opts ...PeriodOption) *PeriodLimit {
	limiter := &PeriodLimit{
		period:     period,
		quota:      quota,
		limitStore: limitStore,
		keyPrefix:  keyPrefix,
	}

	for _, opt := range opts {
		opt(limiter)
	}

	return limiter
}

// Take requests a permit, it returns the permit state.
func (h *PeriodLimit) Take(key string) (int, error) {
	return h.TakeCtx(context.Background(), key)
}

// TakeCtx requests a permit with context, it returns the permit state.
func (h *PeriodLimit) TakeCtx(ctx context.Context, key string) (int, error) {
	resp, err := periodScript.Run(ctx, h.limitStore, []string{h.keyPrefix + key}, []string{
		strconv.Itoa(h.quota),
		strconv.Itoa(h.calcExpireSeconds()),
	}).Result()
	if err != nil {
		logx.Errorf("fail to eval redis script: %v, use in-process limiter for rescue", resp)
		return Unknown, err
	}

	code, ok := resp.(int64)
	if !ok {
		return Unknown, ErrUnknownCode
	}

	switch code {
	case internalOverQuota:
		return OverQuota, nil
	case internalAllowed:
		return Allowed, nil
	case internalHitQuota:
		return HitQuota, nil
	default:
		return Unknown, ErrUnknownCode
	}
}

func (h *PeriodLimit) calcExpireSeconds() int {
	if h.align {
		now := time.Now()
		_, offset := now.Zone()
		unix := now.Unix() + int64(offset)
		return h.period - int(unix%int64(h.period))
	}

	return h.period
}

// Align returns a func to customize a PeriodLimit with alignment.
// For example, if we want to limit end users with 5 sms verification messages every day,
// we need to align with the local timezone and the start of the day.
func Align() PeriodOption {
	return func(l *PeriodLimit) {
		l.align = true
	}
}

func (h *PeriodLimit) Wait(ctx context.Context, key string, maxRetries int, retryInterval time.Duration) error {

	for i := 0; i < maxRetries; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		code, err := h.TakeCtx(ctx, key)
		if err != nil {
			if i == maxRetries-1 {
				return ErrQuotaExhausted
			}
			time.Sleep(retryInterval)
			continue
		}

		switch code {
		case Allowed, HitQuota:
			logx.Info("period limit: allowed maxRetries:", i+1)
			return nil
		case OverQuota:
			if i == maxRetries-1 {
				return ErrQuotaExhausted
			}
		default:
			logx.WithContext(ctx).Errorf("unexpected status code: %d", code)
		}

		time.Sleep(retryInterval)
	}

	return ErrMaxRetriesExceeded
}
