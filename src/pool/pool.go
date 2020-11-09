package pool

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"log"

	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redis_rate/v9"
	"gitlab.com/crypto_project/core/proxypool_service/src/helpers"
	"gitlab.com/crypto_project/core/proxypool_service/src/sources"
)

var proxySingleton *ProxyPool
var ppMux sync.Mutex

var timeBeforeUnhealthyStatusChangePossibleSec int64 = 10 * 60

func newRedisLimiter(ctx *context.Context) *redis_rate.Limiter {

	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")
	addr := host + ":" + port

	log.Printf("Connectiong to redis on %s", addr)

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: os.Getenv("REDIS_PASSWORD"),
		// 0 - use default DB
		DB: 0,
	})

	pingResult := rdb.Ping(*ctx).String()
	log.Printf("Redis: %s \n", pingResult)

	return redis_rate.NewLimiter(rdb)
}

func newProxySingleton() *ProxyPool {
	var proxies [][]string
	helpers.GetProxiesFromENV(&proxies)

	proxyMap := map[int]map[string]*Proxy{}
	currentProxyIndexes := map[int]int{}

	// 0 - max priority (e.g. for trading), 1 - less priority
	proxyMap[0] = map[string]*Proxy{}
	currentProxyIndexes[0] = 0
	proxyMap[1] = map[string]*Proxy{}
	currentProxyIndexes[1] = 0

	limit := 900

	for i, proxyArr := range proxies {
		log.Printf("Init %d proxies with %d priority...", len(proxyArr), i)

		for _, proxyURL := range proxyArr {
			proxyMap[i][proxyURL] = &Proxy{
				Usages:  0,
				Limit:   limit,
				URL:     proxyURL,
				Healthy: true,
			}
		}
	}

	limiterCtx := context.Background()
	redisRateLimiter := newRedisLimiter(&limiterCtx)

	statsd := &sources.StatsdClient{}
	statsd.Init()

	return &ProxyPool{
		Proxies:             proxies,
		CurrentProxyIndexes: currentProxyIndexes,
		ExchangeProxyMap:    proxyMap,
		LimiterCtx:          &limiterCtx,
		RedisRateLimiter:    redisRateLimiter,
		proxyIndexesMux:     sync.Mutex{},
		proxyStatsMux:       sync.Mutex{},
		StartupTime:         time.Now(),
		StatsdMetrics:       statsd,
	}
}

func GetProxyPoolInstance() *ProxyPool {
	ppMux.Lock()
	if proxySingleton == nil {
		log.Printf("Creating new PP singleton...")
		proxySingleton = newProxySingleton()
	}
	ppMux.Unlock()

	return proxySingleton
}

func (pp *ProxyPool) GetProxyByPriority(priority int, weight int) ProxyResponse {
	if pp.Proxies == nil {
		return ProxyResponse{ProxyURL: "", Counter: 0}
	}

	// if next proxy in line marked as unhealthy - get another one
	var currentProxy *Proxy
	for {
		currentProxy = pp.selectProxyByRoundRobin(priority)
		// TODO: CHECK THAT AT LEAST ONE HEALTHY PROXY AVAILABLE
		if currentProxy.Healthy {
			break
		}
	}

	currentProxyURL := currentProxy.URL
	retryCounter := 0

	for {
		// TODO: change currentProxyURL to better key (we have password there, not secure)
		res, redisError := pp.RedisRateLimiter.AllowN(*pp.LimiterCtx, currentProxyURL, redis_rate.PerMinute(currentProxy.Limit), weight)
		if redisError != nil {
			if retryCounter >= 3 {
				log.Printf("Critical error! Failed to serve proxies after %d retries", retryCounter)
				return ProxyResponse{ProxyURL: "", Counter: 0}
			}

			log.Printf("Error while calling AllowN: %s . Retrying...", redisError.Error())
			retryCounter++

			time.Sleep(time.Duration(3*retryCounter) * time.Second)
			continue
		}

		if res.Allowed > 0 {
			break
		} else {
			log.Println("Not allowed. Retry in:", res.RetryAfter)

			if priority == 0 {
				log.Print("Top priority proxy is blocked. Returning low priority proxy.")
				return pp.GetLowPriorityProxy(weight)
			}

			time.Sleep(res.RetryAfter)
		}
	}

	go pp.reportProxyUsage(currentProxy)

	// log.Printf("Returning proxy: %s", currentProxyURL)
	return ProxyResponse{
		ProxyURL: currentProxyURL,
		Counter:  currentProxy.Usages,
	}
}

func (pp *ProxyPool) GetLowPriorityProxy(weight int) ProxyResponse {
	return pp.GetProxyByPriority(1, weight)
}

func (pp *ProxyPool) GetTopPriorityProxy(weight int) ProxyResponse {
	return pp.GetProxyByPriority(0, weight)
}

func (pp *ProxyPool) selectProxyByRoundRobin(priority int) *Proxy {
	// TODO: maybe it's better to use sync.map here
	pp.proxyIndexesMux.Lock()

	currentIndex := pp.CurrentProxyIndexes[priority]
	pp.CurrentProxyIndexes[priority] = currentIndex + 1
	if currentIndex >= len(pp.Proxies[priority]) {
		pp.CurrentProxyIndexes[priority] = 1
		currentIndex = 0
	}

	currentProxyURL := pp.Proxies[priority][currentIndex]
	proxy := pp.ExchangeProxyMap[priority][currentProxyURL]

	pp.proxyIndexesMux.Unlock()

	return proxy
}

func (pp *ProxyPool) MarkProxyAsUnhealthy(proxyPriority int, proxyURL string) {
	if proxiesMap, ok := pp.ExchangeProxyMap[proxyPriority]; ok {
		if proxy, ok := proxiesMap[proxyURL]; ok {
			proxy.Healthy = false
			proxy.HealthStatusLastChange = time.Now().Unix()
			log.Printf("Proxy with URL %s marked as unhealthy (%d priority)", proxyURL, proxyPriority)
		} else {
			log.Printf("Error. No proxy with URL %s found (%d priority)", proxyURL, proxyPriority)
		}
	} else {
		log.Printf("Error. No proxies with %d priority", proxyPriority)
	}
}

func (pp *ProxyPool) MarkProxyAsHealthy(proxyPriority int, proxyURL string) {
	if proxiesMap, ok := pp.ExchangeProxyMap[proxyPriority]; ok {
		if proxy, ok := proxiesMap[proxyURL]; ok {
			currentUnixTimestamp := time.Now().Unix()

			if proxy.Healthy == false && currentUnixTimestamp > proxy.HealthStatusLastChange+timeBeforeUnhealthyStatusChangePossibleSec {
				proxy.Healthy = true
				proxy.HealthStatusLastChange = currentUnixTimestamp
				log.Printf("Proxy with URL %s marked as healthy (%d priority)", proxyURL, proxyPriority)
			}
		} else {
			log.Printf("Error. No proxy with URL %s found (%d priority)", proxyURL, proxyPriority)
		}
	} else {
		log.Printf("Error. No proxies with %d priority", proxyPriority)
	}
}

func (pp *ProxyPool) GetStats() []string {
	stats := []string{}
	timeSinceStartup := time.Since(pp.StartupTime).Seconds()

	for priority := range pp.ExchangeProxyMap {
		for _, proxy := range pp.ExchangeProxyMap[priority] {
			proxyIP := helpers.FindIP(proxy.URL)
			data := fmt.Sprintf("Proxy %s with priority %d got %f requests/sec on avg \n", proxyIP, priority, float64(proxy.Usages)/timeSinceStartup)
			stats = append(stats, data)
		}
	}

	return stats
}

func (pp *ProxyPool) reportProxyUsage(proxy *Proxy) {
	pp.proxyStatsMux.Lock()
	proxy.Usages++
	pp.proxyStatsMux.Unlock()
}
