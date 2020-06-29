package pool

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"go.uber.org/ratelimit"
)

type Limit struct {
	requests int64
	overPeriod int64
}


type Proxy struct {
	RateLimiter ratelimit.Limiter
}

type ProxyPool struct{
	Proxies [][] string
	CurrentProxyIndexes map[int] int
	LimitMap map[string] *Limit // Exchange -> Proxy -> Requests Made
	ExchangeProxyMap map[int]map[string] *Proxy // Exchange -> Proxy -> Requests Made
}

// NewSignalSingleton returns SignalSingleton instance
func newProxySingleton() *ProxyPool {
	proxiesBASE64 := os.Getenv("PROXYLIST")
	proxiesJSON, err := base64.StdEncoding.DecodeString(proxiesBASE64)
	if err != nil {
		fmt.Println("error:", err)
		return nil
	}
	var proxies [][] string

	json.Unmarshal([]byte(proxiesJSON), &proxies)

	// exchanges := [2]string{"binance", "bittrex"}

	proxyMap := map[int]map[string]*Proxy{}
	currentProxyIndexes := map[int]int{}

	// 0 - max priority (e.g. for tgrading), 1 - less priority

	proxyMap[0] = map[string]*Proxy{}
	currentProxyIndexes[0] = 0
	proxyMap[1] = map[string]*Proxy{}
	currentProxyIndexes[1] = 0

	for i, proxyArr := range proxies {
		for _, proxy := range proxyArr {
			proxyMap[i][proxy] = &Proxy{
				RateLimiter: ratelimit.New(4), // 240 / min
			}
		}
	}

	// env PROXYLIST
	return &ProxyPool{
		Proxies:proxies,
		CurrentProxyIndexes: currentProxyIndexes,
		ExchangeProxyMap: proxyMap,
	}
}

var proxySingleton *ProxyPool

func GetProxyPoolInstance() *ProxyPool {
	if proxySingleton == nil {
		proxySingleton = newProxySingleton()
	}
	return proxySingleton
}


func (pp *ProxyPool) GetProxyByPriority(priority int) string {
	currentIndex := pp.CurrentProxyIndexes[priority]
	pp.CurrentProxyIndexes[priority] = currentIndex + 1
	if currentIndex >= len(pp.Proxies) {
		pp.CurrentProxyIndexes[priority] = 1
		currentIndex = 0
	}

	currentProxy := pp.Proxies[priority][currentIndex]

	// currentTime := time.Now().UnixNano()
	currentRequests := pp.ExchangeProxyMap[priority][currentProxy]
	_ = currentRequests.RateLimiter.Take()


	return currentProxy
}

