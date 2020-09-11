package pool

import (
	"context"
	"os"
	"strconv"
	"sync"
	"time"

	"log"

	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redis_rate/v9"
)

var proxySingleton *ProxyPool

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

	_ = rdb.FlushDB(*ctx).Err()

	return redis_rate.NewLimiter(rdb)
}

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

	normalLimit := 600

	for i, proxyArr := range proxies {
		log.Printf("Init %d proxies with %d priority...", len(proxyArr), i)

		for _, proxy := range proxyArr {
			proxyMap[i][proxy] = &Proxy{
				Usages:        0,
				Limit:         normalLimit,
				Locked:        false,
				NeedResponses: 0,
				Name:          proxy,
			}
		}
	}

	limiterCtx := context.Background()
	redisRateLimiter := newRedisLimiter(&limiterCtx)

	return &ProxyPool{
		Proxies:             proxies,
		CurrentProxyIndexes: currentProxyIndexes,
		ExchangeProxyMap:    proxyMap,
		DebtorsMap:          map[string]time.Time{},
		LimiterCtx:          &limiterCtx,
		RedisRateLimiter:    redisRateLimiter,
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
	pp.proxyIndexesMux.Unlock()

	for {
		res, err := pp.RedisRateLimiter.AllowN(*pp.LimiterCtx, currentProxyURL, redis_rate.PerMinute(currentProxy.Limit), weight)
		if err != nil {
			log.Printf("Error while calling AllowN: %s", err.Error())
			time.Sleep(3 * time.Second)
		}

		if res.Allowed > 0 {
			break
		} else {
			log.Println("Allowed:", res.Allowed, "Remaining:", res.Remaining, "Retry in:", res.RetryAfter)

			if priority == 0 {
				log.Print("Top priority proxy is blocked. Returning low priority proxy.")
				return pp.GetLowPriorityProxy(weight)
			}

			time.Sleep(res.RetryAfter)
		}
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
