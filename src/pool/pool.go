package pool

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"go.uber.org/ratelimit"
	"log"
	"os"
)

type Limit struct {
	requests   int64
	overPeriod int64
}

type Proxy struct {
	RateLimiter ratelimit.Limiter
}

type ProxyPool struct {
	Proxies             [][]string
	CurrentProxyIndexes map[int]int
	LimitMap            map[string]*Limit         // Exchange -> Proxy -> Requests Made
	ExchangeProxyMap    map[int]map[string]*Proxy // Exchange -> Proxy -> Requests Made
}

func newProxySingleton() *ProxyPool {
	proxiesBASE64 := os.Getenv("PROXYLIST")
	proxiesJSON, err := base64.StdEncoding.DecodeString(proxiesBASE64)
	if err != nil {
		fmt.Println("error:", err)
		return nil
	}
	var proxies [][]string
	json.Unmarshal([]byte(proxiesJSON), &proxies)

	proxyMap := map[int]map[string]*Proxy{}
	currentProxyIndexes := map[int]int{}

	// 0 - max priority (e.g. for tgrading), 1 - less priority

	proxyMap[0] = map[string]*Proxy{}
	currentProxyIndexes[0] = 0
	proxyMap[1] = map[string]*Proxy{}
	currentProxyIndexes[1] = 0

	for i, proxyArr := range proxies {
		log.Printf("Init %d proxies with %d priority...", len(proxyArr), i)

		for _, proxy := range proxyArr {
			proxyMap[i][proxy] = &Proxy{
				RateLimiter: ratelimit.New(4), // 240 / min
			}
		}
	}

	return &ProxyPool{
		Proxies:             proxies,
		CurrentProxyIndexes: currentProxyIndexes,
		ExchangeProxyMap:    proxyMap,
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
	log.Printf("Got GetProxyByPriority request with %d priority", priority)

	currentIndex := pp.CurrentProxyIndexes[priority]
	pp.CurrentProxyIndexes[priority] = currentIndex + 1
	println("currentIndex", currentIndex, "len proxies", len(pp.Proxies[priority]))
	if currentIndex >= len(pp.Proxies[priority]) {
		pp.CurrentProxyIndexes[priority] = 1
		currentIndex = 0
	}

	currentProxy := pp.Proxies[priority][currentIndex]

	// currentTime := time.Now().UnixNano()
	currentRequests := pp.ExchangeProxyMap[priority][currentProxy]
	_ = currentRequests.RateLimiter.Take()

	return currentProxy
}
