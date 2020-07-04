package tests

import (
	"log"
	"sync"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"gitlab.com/crypto_project/core/proxypool_service/src/pool"
)

func TestProxyDelay(t *testing.T) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	proxyPool := pool.GetProxyPoolInstance()

	iterations := 90
	rateLimit := 3
	proxyPriority := 1
	numberOfProxies := 2 // based on tests/.env file

	// we use wait group to await ALL requests to be finished
	var wg sync.WaitGroup
	wg.Add(iterations)

	start := time.Now()
	for i := 0; i < iterations; i++ {
		// all request are done simultaneously to imitate heavy load
		go getProxyUsingWaitGroup(proxyPool, proxyPriority, &wg)
	}

	wg.Wait()
	duration := time.Since(start).Milliseconds()
	expectedExecTime := ((float64(iterations)/float64(rateLimit))/float64(numberOfProxies) - (1.0 / float64(rateLimit))) * 1000

	log.Printf("Total %d requests to pool with %d proxies and rate limit %d req/s done in %d ms, expected ~ %f ms", iterations, numberOfProxies, rateLimit, duration, expectedExecTime)

	if duration < int64(expectedExecTime) {
		t.Error("Not enough time passed, something is wrong with rate limiter.")
	}
}

func getProxyUsingWaitGroup(proxyPool *pool.ProxyPool, priority int, wg *sync.WaitGroup) {
	proxyURL := proxyPool.GetProxyByPriority(priority)
	_ = proxyURL
	log.Printf("Got proxy URL: %s", proxyURL)
	wg.Done()
}
