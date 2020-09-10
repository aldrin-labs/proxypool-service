package pool

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type Proxy struct {
	RateLimiter   *rate.Limiter
	Usages        int
	Name          string
	Locked        bool
	Limit         float64
	NeedResponses int
}

type ProxyResponse struct {
	Proxy   string `json:"proxy"`
	Counter int    `json:"counter"`
}

type ProxyPool struct {
	Proxies             [][]string
	CurrentProxyIndexes map[int]int
	ExchangeProxyMap    map[int]map[string]*Proxy
	DebtorsMap          map[string]time.Time

	StartupTime     time.Time
	Timeout         int
	proxyIndexesMux sync.Mutex
	proxyStatsMux   sync.Mutex
}
