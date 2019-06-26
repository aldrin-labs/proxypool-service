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
	Proxies [] string
	CurrentProxyIndex int
	LimitMap map[string] *Limit // Exchange -> Proxy -> Requests Made
	ExchangeProxyMap map[string]map[string] *Proxy // Exchange -> Proxy -> Requests Made
}

// NewSignalSingleton returns SignalSingleton instance
func newProxySingleton() *ProxyPool {
	proxiesBASE64 := os.Getenv("PROXYLIST")
	proxiesJSON, err := base64.StdEncoding.DecodeString(proxiesBASE64)
	if err != nil {
		fmt.Println("error:", err)
		return nil
	}
	var proxies [] string

	json.Unmarshal([]byte(proxiesJSON), &proxies)

	// exchanges := [2]string{"binance", "bittrex"}

	proxyMap := map[string]map[string]*Proxy{}

	proxyMap["binance"] = map[string]*Proxy{}
	proxyMap["bittrex"] = map[string]*Proxy{}

	for _, proxy := range proxies {
		proxyMap["binance"][proxy] = &Proxy{
			RateLimiter: ratelimit.New(4), // 80 / min
		}
		proxyMap["bittrex"][proxy] = &Proxy{
			RateLimiter: ratelimit.New(1), // 57 / min
		}
	}

	// env PROXYLIST
	return &ProxyPool{
		Proxies:proxies,
		CurrentProxyIndex: 0,
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


func (pp *ProxyPool) GetProxyByExchange(exchangeName string) string {
	currentIndex := pp.CurrentProxyIndex
	pp.CurrentProxyIndex = pp.CurrentProxyIndex + 1
	if pp.CurrentProxyIndex >= len(pp.Proxies) {
		pp.CurrentProxyIndex = 0
	}
	if currentIndex >= len(pp.Proxies) {
		currentIndex = 0
	}

	currentProxy := pp.Proxies[currentIndex]

	// currentTime := time.Now().UnixNano()
	currentRequests := pp.ExchangeProxyMap[exchangeName][currentProxy]
	_ = currentRequests.RateLimiter.Take()


	return currentProxy
}

