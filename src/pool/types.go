package pool

import (
	"context"
	"sync"
	"time"

	"github.com/go-redis/redis_rate/v9"
	"gitlab.com/crypto_project/core/proxypool_service/src/sources"
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

	StatsdMetrics *sources.StatsdClient
}

type Proxy struct {
	URL                    string
	Usages                 int
	Limit                  int
	Healthy                bool
	HealthStatusLastChange int64
}

type ProxyResponse struct {
	ProxyURL string `json:"proxy"`
	Counter  int    `json:"counter"`
}
