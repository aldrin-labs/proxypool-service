package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"gitlab.com/crypto_project/core/proxypool_service/src/api"
)

// this test requires redis connection

func TestHTTPRequestThrottling(t *testing.T) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	go api.RunServer()

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

	start := time.Now()
	for i := 0; i < threads; i++ {
		// all request are done simultaneously from multiple threads to imitate heavy load
		go makeProxyRequests(proxyPriority, requestWeight, requestsByThread, &wg)
	}

	requestDuration := time.Since(start).Milliseconds()
	log.Printf("Test started %d threads in %d ms...", threads, requestDuration)

	wg.Wait()

	duration := time.Since(start).Milliseconds()

	thresholdWeightInMin := 60 * rateLimitSec * numberOfProxies
	totalWeight := requestWeight * threads * requestsByThread
	overThresholdWeight := totalWeight - thresholdWeightInMin
	if overThresholdWeight < 0 {
		overThresholdWeight = 0
	}

	expectedExecTime := (float64(overThresholdWeight) / float64(totalWeight)) * 60 * 1000

	log.Print("thresholdWeightInMin: ", thresholdWeightInMin, " overThresholdWeight: ", overThresholdWeight)
	log.Printf("Total %d requests with weight %d to pool with %d proxies and rate limit %d req/s done in %d ms, expected > ~ %f ms", totalRequests, totalWeight, numberOfProxies, rateLimitSec, duration, expectedExecTime)

	if duration < int64(expectedExecTime) {
		t.Error("Not enough time passed, something is wrong with rate limiter.")
	}
}

func makeProxyRequests(priority int, weight int, numberOfRequests int, wg *sync.WaitGroup) {
	for i := 0; i < numberOfRequests; i++ {
		proxyPoolURL := fmt.Sprintf("http://localhost:5901/getProxy?priority=%d&weight=%d", priority, weight)
		proxyDataInterface := makeHTTPRequest(proxyPoolURL)
		proxyDataString := fmt.Sprintf("%s", proxyDataInterface)
		// log.Printf("Got proxy string: %s", proxyDataString)

		proxyData := &struct {
			Proxy   string `json:"proxy"`
			Counter int    `json:"counter"`
		}{}

		json.Unmarshal([]byte(proxyDataString), proxyData)
		// log.Printf("Got proxy data: %v", proxyData.Proxy)

		wg.Done()
	}
}

func makeHTTPRequest(url string) interface{} {
	req, err := http.NewRequest("GET", url, bytes.NewBuffer([]byte{}))
	if err != nil {
		log.Println("Request error", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Request error", err)
	}

	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	return body
}
