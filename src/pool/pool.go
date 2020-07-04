package pool

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"sync"

	"log"

	"golang.org/x/time/rate"
)

type Proxy struct {
	RateLimiter *rate.Limiter
}

type ProxyPool struct {
	Proxies             [][]string
	CurrentProxyIndexes map[int]int
	ExchangeProxyMap    map[int]map[string]*Proxy
	proxyIndexesMux     sync.Mutex
}

var proxySingleton *ProxyPool

func newProxySingleton() *ProxyPool {
	var proxies [][]string
	getProxiesFromENV(&proxies)

	proxyMap := map[int]map[string]*Proxy{}
	currentProxyIndexes := map[int]int{}

	// 0 - max priority (e.g. for trading), 1 - less priority
	proxyMap[0] = map[string]*Proxy{}
	currentProxyIndexes[0] = 0
	proxyMap[1] = map[string]*Proxy{}
	currentProxyIndexes[1] = 0

	normalLimit := rate.Limit(3) // 180 / min
	burst := 3

	for i, proxyArr := range proxies {
		log.Printf("Init %d proxies with %d priority...", len(proxyArr), i)

		for _, proxy := range proxyArr {
			proxyMap[i][proxy] = &Proxy{
				RateLimiter: rate.NewLimiter(normalLimit, burst),
			}
		}
	}

	return &ProxyPool{
		Proxies:             proxies,
		CurrentProxyIndexes: currentProxyIndexes,
		ExchangeProxyMap:    proxyMap,
		proxyIndexesMux:     sync.Mutex{},
	}
}

func GetProxyPoolInstance() *ProxyPool {
	if proxySingleton == nil {
		proxySingleton = newProxySingleton()
	}
	return proxySingleton
}

func getProxiesFromENV(proxies *[][]string) {
	proxiesBASE64 := os.Getenv("PROXYLIST")
	proxiesJSON, err := base64.StdEncoding.DecodeString(proxiesBASE64)
	if err != nil {
		log.Print("error:", err)
		return
	}

	jsonErr := json.Unmarshal([]byte(proxiesJSON), proxies)
	if jsonErr != nil {
		log.Print("json error:", jsonErr)
		return
	}
}

func (pp *ProxyPool) GetProxyByPriority(priority int) string {
	if pp.Proxies == nil {
		return ""
	}

	// TODO: maybe it's better to use sync.map here
	pp.proxyIndexesMux.Lock()
	currentIndex := pp.CurrentProxyIndexes[priority]
	pp.CurrentProxyIndexes[priority] = currentIndex + 1
	if currentIndex >= len(pp.Proxies[priority]) {
		pp.CurrentProxyIndexes[priority] = 1
		currentIndex = 0
	}

	currentProxyURL := pp.Proxies[priority][currentIndex]
	currentProxyRateLimiter := pp.ExchangeProxyMap[priority][currentProxyURL].RateLimiter
	pp.proxyIndexesMux.Unlock()

	if currentProxyRateLimiter.Allow() == false {
		if priority == 0 {
			log.Print("Top priority proxy is blocked. Returning low priority proxy.")
			return pp.GetLowPriorityProxy()
		}

		ctx := context.Background()
		currentProxyRateLimiter.Wait(ctx)
	}

	return currentProxyURL
}

func (pp *ProxyPool) GetLowPriorityProxy() string {
	return pp.GetProxyByPriority(1)
}

func (pp *ProxyPool) GetTopPriorityProxy() string {
	return pp.GetProxyByPriority(0)
}
