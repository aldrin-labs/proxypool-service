package pool

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	loggly_client "gitlab.com/crypto_project/core/proxypool_service/src/sources/loggly"

	"github.com/go-errors/errors"
	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redis_rate/v9"
	"gitlab.com/crypto_project/core/proxypool_service/src/helpers"
	"gitlab.com/crypto_project/core/proxypool_service/src/sources"
)

var proxySingleton *ProxyPool
var ppMux sync.Mutex

var timeBeforeUnhealthyStatusChangePossibleSec int64 = 3 * 60

func newRedisLimiter(ctx *context.Context) *redis_rate.Limiter {

	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")
	addr := host + ":" + port

	loggly_client.GetInstance().Infof("Connectiong to redis on %s", addr)

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: os.Getenv("REDIS_PASSWORD"),
		// 0 - use default DB
		DB: 0,
	})

	pingResponse := rdb.Ping(*ctx).String()
	loggly_client.GetInstance().Info("Redis:", pingResponse)
	if strings.Contains(pingResponse, "error") || strings.Contains(pingResponse, "timeout") || strings.Contains(pingResponse, "refused") {
		loggly_client.GetInstance().Fatal("Redis connection error. Exiting...")
	}

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
		loggly_client.GetInstance().Infof("Init %d proxies with %d priority...", len(proxyArr), i)

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
		loggly_client.GetInstance().Infof("Creating new PP singleton...")
		proxySingleton = newProxySingleton()
	}
	ppMux.Unlock()

	return proxySingleton
}

func (pp *ProxyPool) GetProxyByPriority(priority int, weight int) ProxyResponse {
	if pp.Proxies == nil {
		pp.StatsdMetrics.Inc("pool.empty_proxy_returned")
		return ProxyResponse{ProxyURL: "", Counter: 0}
	}

	currentProxy, err := pp.SelectProxy(priority)
	if err != nil {
		pp.StatsdMetrics.Inc("pool.empty_proxy_returned")
		return ProxyResponse{ProxyURL: "", Counter: 0}
	}

	pp.StatsdMetrics.IncBy("pool.proxy_weight_used", int64(weight))
	if weight > 5 {
		log.Printf("registered request with weigth > 5 : %d priority: %d", weight, priority)
	}

	currentProxyURL := currentProxy.URL
	retryCounter := 0

	for {
		// make request to redis rate limiter
		startRequestToRedis := time.Now()
		// TODO: change currentProxyURL to better key (we have password there, not secure)
		res, redisError := pp.RedisRateLimiter.AllowN(*pp.LimiterCtx, currentProxyURL, redis_rate.PerMinute(currentProxy.Limit), weight)
		requestToRedisDuration := time.Since(startRequestToRedis)
		pp.GetMetricsClient().Timing("pool.redis_rate_limiter_call.duration", int64(requestToRedisDuration.Milliseconds()))

		if redisError != nil {
			if retryCounter >= 3 {
				loggly_client.GetInstance().Infof("Critical error! Failed to serve proxies after %d retries", retryCounter)
				pp.StatsdMetrics.Inc("pool.empty_proxy_returned")
				return ProxyResponse{ProxyURL: "", Counter: 0}
			}

			loggly_client.GetInstance().Infof("Error while calling AllowN: %s . Retrying...", redisError.Error())
			pp.StatsdMetrics.Inc("pool.redis_error")
			retryCounter++

			time.Sleep(time.Duration(3*retryCounter) * time.Second)
			continue
		}

		if res.Allowed > 0 {
			// if we can return proxy immediately
			break
		} else {
			// if proxy is over rate limit we should throttle request
			loggly_client.GetInstance().Info("All proxies are busy. Throttling for:", res.RetryAfter)

			if priority == 0 {
				loggly_client.GetInstance().Info("Top priority proxy is blocked. Returning low priority proxy.")
				pp.StatsdMetrics.Inc("pool.lower_priority_proxy_switch")
				return pp.GetLowPriorityProxy(weight)
			}

			pp.StatsdMetrics.Inc("pool.throttled")
			time.Sleep(res.RetryAfter)
		}
	}

	go pp.reportProxyUsage(currentProxy)

	// loggly_client.GetInstance().Infof("Returning proxy: %s", currentProxyURL)
	pp.StatsdMetrics.Inc("pool.proxy_served")
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
			loggly_client.GetInstance().Infof("Proxy with URL %s marked as unhealthy (%d priority)", proxyURL, proxyPriority)
			pp.StatsdMetrics.Inc("pool.marked_as_unhealthy")
		} else {
			loggly_client.GetInstance().Infof("Error. No proxy with URL %s found (%d priority)", proxyURL, proxyPriority)
		}
	} else {
		loggly_client.GetInstance().Infof("Error. No proxies with %d priority", proxyPriority)
	}
}

func (pp *ProxyPool) MarkProxyAsHealthy(proxyPriority int, proxyURL string) {
	if proxiesMap, ok := pp.ExchangeProxyMap[proxyPriority]; ok {
		if proxy, ok := proxiesMap[proxyURL]; ok {
			currentUnixTimestamp := time.Now().Unix()

			if proxy.Healthy == false && currentUnixTimestamp > proxy.HealthStatusLastChange+timeBeforeUnhealthyStatusChangePossibleSec {
				proxy.Healthy = true
				proxy.HealthStatusLastChange = currentUnixTimestamp
				loggly_client.GetInstance().Infof("Proxy with URL %s marked as healthy (%d priority)", proxyURL, proxyPriority)
				pp.StatsdMetrics.Inc("pool.marked_as_healthy")
			}
		} else {
			loggly_client.GetInstance().Infof("Error. No proxy with URL %s found (%d priority)", proxyURL, proxyPriority)
		}
	} else {
		loggly_client.GetInstance().Infof("Error. No proxies with %d priority", proxyPriority)
	}
}

func (pp *ProxyPool) SelectProxy(priority int) (*Proxy, error) {
	currentProxy := &Proxy{}

	var retries = 0
	for {
		atLeastOneProxyIsHealthy := pp.AtLeastOneProxyIsHealthy(priority)
		if atLeastOneProxyIsHealthy {
			currentProxy = pp.selectProxyByRoundRobin(priority)
		} else {
			if priority == 0 {
				// try proxy with lower priority
				currentProxy = pp.selectProxyByRoundRobin(1)
			} else {
				// just wait, nothing more to do, all proxies are unhealthy
				time.Sleep(10 * time.Second)
			}
		}

		// if next proxy in line marked as unhealthy - get another one
		if currentProxy.Healthy {
			break
		}

		if retries > 10 {
			return nil, errors.New("Failed to select proxy after number of retries")
		}
		retries++
	}
	return currentProxy, nil
}

func (pp *ProxyPool) AtLeastOneProxyIsHealthy(priority int) bool {
	for _, proxy := range pp.ExchangeProxyMap[priority] {
		if proxy.Healthy {
			return true
		}
	}
	return false
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

func (pp *ProxyPool) GetMetricsClient() *sources.StatsdClient {
	return pp.StatsdMetrics
}

func (pp *ProxyPool) reportProxyUsage(proxy *Proxy) {
	pp.proxyStatsMux.Lock()
	proxy.Usages++
	pp.proxyStatsMux.Unlock()
}
