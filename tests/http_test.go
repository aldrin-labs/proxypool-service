package tests

import (
	"log"
	"sync"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"gitlab.com/crypto_project/core/proxypool_service/src/api"
	"gitlab.com/crypto_project/core/proxypool_service/tests/tests_helpers"
)

// this test requires redis connection

func TestHTTPRequestThrottling(t *testing.T) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	port := ":5901"
	go api.RunServer(port)

	// waiting for api server to go up
	time.Sleep(1 * time.Second)

	threads := 10
	requestsByThread := 34
	totalRequests := threads * requestsByThread
	proxyPriority := 1
	requestWeight := 4
	rateLimitSec := 10
	numberOfProxies := 2 // based on tests/.env file

	// we use wait group to await ALL requests to be finished
	var wg sync.WaitGroup
	wg.Add(totalRequests)

	// make requests to proxypool
	start := time.Now()
	for i := 0; i < threads; i++ {
		// all request are done simultaneously from multiple threads to imitate heavy load
		go tests_helpers.MakeProxyRequests(port, proxyPriority, requestWeight, requestsByThread, &wg)
	}

	requestDuration := time.Since(start).Milliseconds()
	log.Printf("Test started %d threads in %d ms...", threads, requestDuration)

	// wait and measure execution time
	wg.Wait()
	duration := time.Since(start).Milliseconds()

	// calculate expected execution time
	thresholdWeightInMin := 60 * rateLimitSec * numberOfProxies
	totalWeight := requestWeight * threads * requestsByThread
	overThresholdWeight := totalWeight - thresholdWeightInMin
	if overThresholdWeight < 0 {
		overThresholdWeight = 0
	}

	expectedExecTime := (float64(overThresholdWeight) / float64(totalWeight)) * 60 * 1000

	// log.Print("thresholdWeightInMin: ", thresholdWeightInMin, " overThresholdWeight: ", overThresholdWeight)
	log.Printf("Total %d requests with weight %d to pool with %d proxies and rate limit %d req/s done in %d ms, expected > ~ %f ms", totalRequests, totalWeight, numberOfProxies, rateLimitSec, duration, expectedExecTime)

	if duration < int64(expectedExecTime) {
		t.Error("Not enough time passed, something is wrong with rate limiter.")
	}
}
