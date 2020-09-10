package pool

import (
	"context"
	"strconv"
	"sync"
	"time"

	"log"

	"golang.org/x/time/rate"
)

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

	normalLimit := 500.0 / 60.0 // 500 / min
	normalRateLimit := rate.Limit(normalLimit)
	// how much requests can be run simultaneously if there were no throttling when they were received
	burst := 110

	for i, proxyArr := range proxies {
		log.Printf("Init %d proxies with %d priority...", len(proxyArr), i)

		for _, proxy := range proxyArr {
			proxyMap[i][proxy] = &Proxy{
				RateLimiter:   rate.NewLimiter(normalRateLimit, burst),
				Usages:        0,
				Limit:         normalLimit,
				Locked:        false,
				NeedResponses: 0,
				Name:          proxy,
			}
		}
	}

	return &ProxyPool{
		Proxies:             proxies,
		CurrentProxyIndexes: currentProxyIndexes,
		ExchangeProxyMap:    proxyMap,
		DebtorsMap:          map[string]time.Time{},
		proxyIndexesMux:     sync.Mutex{},
		proxyStatsMux:       sync.Mutex{},
		StartupTime:         time.Now(),
		Timeout:             90,
	}
}

func GetProxyPoolInstance() *ProxyPool {
	if proxySingleton == nil {
		proxySingleton = newProxySingleton()
		//go proxySingleton.CheckProxyTimeout()
	}
	return proxySingleton
}

func (pp *ProxyPool) GetProxyByPriority(priority int, weight int) ProxyResponse {
	if pp.Proxies == nil {
		return ProxyResponse{Proxy: "", Counter: 0}
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
	currentProxy := pp.ExchangeProxyMap[priority][currentProxyURL]
	currentProxyRateLimiter := currentProxy.RateLimiter
	pp.proxyIndexesMux.Unlock()

	if currentProxyRateLimiter.AllowN(time.Now(), weight) == false {
		if priority == 0 {
			log.Print("Top priority proxy is blocked. Returning low priority proxy.")
			return pp.GetLowPriorityProxy(weight)
		}

		ctx := context.Background()
		currentProxyRateLimiter.WaitN(ctx, weight)
	}

	pp.proxyStatsMux.Lock()
	currentProxy.Usages++
	currentProxy.NeedResponses++
	pp.DebtorsMap[currentProxyURL+"_"+strconv.Itoa(currentProxy.Usages)] = time.Now()
	pp.proxyStatsMux.Unlock()

	log.Print("return proxy url: ", currentProxyURL, " proxy, needResponses: ", currentProxy.NeedResponses)
	return ProxyResponse{
		Proxy:   currentProxyURL,
		Counter: currentProxy.Usages,
	}
}

func (pp *ProxyPool) GetLowPriorityProxy(weight int) ProxyResponse {
	return pp.GetProxyByPriority(1, weight)
}

func (pp *ProxyPool) GetTopPriorityProxy(weight int) ProxyResponse {
	return pp.GetProxyByPriority(0, weight)
}
