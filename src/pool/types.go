package pool

import (
	"context"
	"sync"
	"time"

	"github.com/go-redis/redis_rate/v9"
)

type ProxyPool struct {
	Proxies             [][]string
	CurrentProxyIndexes map[int]int
	ExchangeProxyMap    map[int]map[string]*Proxy

	LimiterCtx       *context.Context
	RedisRateLimiter *redis_rate.Limiter

	StartupTime     time.Time
	proxyIndexesMux sync.Mutex
	proxyStatsMux   sync.Mutex
}

type Proxy struct {
	URL    string
	Usages int
	Limit  int
}

type ProxyResponse struct {
	ProxyURL string `json:"proxy"`
	Counter  int    `json:"counter"`
}
