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
	DebtorsMap          map[string]time.Time

	LimiterCtx       *context.Context
	RedisRateLimiter *redis_rate.Limiter

	StartupTime     time.Time
	Timeout         int
	proxyIndexesMux sync.Mutex
	proxyStatsMux   sync.Mutex
}

type Proxy struct {
	Usages        int
	Name          string
	Locked        bool
	Limit         int
	NeedResponses int
}

type ProxyResponse struct {
	Proxy   string `json:"proxy"`
	Counter int    `json:"counter"`
}
