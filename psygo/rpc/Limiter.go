package rpc

import (
	"golang.org/x/time/rate"
	"time"
)

type Limiter struct {
	LimiterInstance *rate.Limiter
	LimiterTimeOut  time.Duration
}
