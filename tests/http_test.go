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

func TestHTTPRequestThrottling(t *testing.T) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	go api.RunServer()

	// waiting for api server to go up
	time.Sleep(1 * time.Second)

	threads := 30
	requestsByThread := 2
	totalRequests := threads * requestsByThread
	proxyPriority := 1
	requestWeight := 10
	rateLimit := 3
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
	expectedExecTime := ((float64(totalRequests)/float64(rateLimit))/float64(numberOfProxies) - (1.0 / float64(rateLimit))) * 1000

	log.Printf("Total %d requests to pool with %d proxies and rate limit %d req/s done in %d ms, expected ~ %f ms", totalRequests, numberOfProxies, rateLimit, duration, expectedExecTime)

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

		// send exempt request
		proxyPoolExemptURL := "http://localhost:5901/exempt"
		log.Printf("Making request to: %s", proxyPoolExemptURL)
		makeHTTPPostRequest(proxyPoolExemptURL, proxyData.Proxy, proxyData.Counter)

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

func makeHTTPPostRequest(requrl string, proxy string, counter int) interface{} {
	d := fmt.Sprintf("{\"proxy\":\"%s\",\"counter\":\"%d\"}", proxy, counter)
	data := []byte(d)
	r := bytes.NewReader(data)

	req, err := http.NewRequest("POST", requrl, r) //strings.NewReader(data.Encode())
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
