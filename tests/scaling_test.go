package tests

import (
	"log"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"gitlab.com/crypto_project/core/proxypool_service/src/api"
	"gitlab.com/crypto_project/core/proxypool_service/tests/tests_helpers"
)

func TestScaling(t *testing.T) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// simulating multiple service instances
	portOne := ":5901"
	go api.RunServer(portOne)

	portTwo := ":5902"
	go api.RunServer(portTwo)

	portThree := ":5903"
	go api.RunServer(portThree)

	// waiting for api servers to go up
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

	start := time.Now()
	for i := 0; i < threads; i++ {
		// all request are done simultaneously from multiple threads to imitate heavy load
		port := selectRandomServicePort(portOne, portTwo, portThree)

		go tests_helpers.MakeProxyRequests(port, proxyPriority, requestWeight, requestsByThread, &wg)
	}

	requestDuration := time.Since(start).Milliseconds()
	log.Printf("Test started %d threads in %d ms...", threads, requestDuration)

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

func selectRandomServicePort(portOne string, portTwo string, portThree string) string {
	randomInt := rand.Intn(3)
	switch randomInt {
	case 0:
		return portOne
	case 1:
		return portTwo
	case 2:
		return portThree
	default:
		return portOne
	}
}
