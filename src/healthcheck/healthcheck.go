package healthcheck

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"gitlab.com/crypto_project/core/proxypool_service/src/helpers"
	"gitlab.com/crypto_project/core/proxypool_service/src/pool"
	"gitlab.com/crypto_project/core/proxypool_service/src/sources"
)

var healthcheckInterval = 20 * time.Second

func CheckProxy(proxyURL string, priority int, ch chan<- HealthCheckResponse) {
	binanceFapiTimeEndpoint := "https://fapi.binance.com/fapi/v1/time"
	binanceSpotEndpoint := "https://api.binance.com/api/v3/exchangeInfo"

	realIP, country := getProxyInfo(proxyURL)

	start := time.Now()
	rawResult, futuresHeaders := helpers.MakeHTTPRequestUsingProxy(binanceFapiTimeEndpoint, proxyURL)
	duration := time.Since(start)
	_, spotHeaders := helpers.MakeHTTPRequestUsingProxy(binanceSpotEndpoint, proxyURL)

	usedWeightFutures := futuresHeaders.Get("X-MBX-USED-WEIGHT-1m")
	usedWeightSpot := spotHeaders.Get("X-MBX-USED-WEIGHT-1m")

	result := BinancePingResponse{}
	hcResponse := HealthCheckResponse{
		Success:           false,
		UsedSpotWeight:    usedWeightSpot,
		UsedFuturesWeight: usedWeightFutures,
		ProxyURL:          proxyURL,
		ProxyPriority:     priority,
		ProxyRealIP:       realIP,
		ProxyCountry:      country,
		ResponseTimeMs:    duration.Milliseconds(),
		Response:          &result,
	}

	jsonErr := json.Unmarshal(rawResult.([]byte), &result)
	if jsonErr != nil {
		log.Print("Json decode error:", rawResult)
		ch <- hcResponse
		return
	}

	if result.ServerTime > 0 {
		hcResponse.Success = true
		ch <- hcResponse
		return
	}

	ch <- hcResponse
	return
}

func getProxyInfo(proxyURL string) (string, string) {
	ipCheckEndpoint := "https://api.myip.com"

	result := IPCheckResponse{}

	rawResult, _ := helpers.MakeHTTPRequestUsingProxy(ipCheckEndpoint, proxyURL)

	jsonErr := json.Unmarshal(rawResult.([]byte), &result)
	if jsonErr != nil {
		log.Printf("Json decode error: %s", jsonErr.Error())
	}

	return result.IP, result.Country
}

// warning, this call to binance is not counted in redis rate limiter
func RunProxiesHealthcheck() {
	time.Sleep(3 * time.Second)
	for {
		log.Printf("Starting proxy healthcheck...")

		results := make(map[string]HealthCheckResponse)

		ch := make(chan HealthCheckResponse)

		pp := pool.GetProxyPoolInstance()
		proxies := pp.Proxies
		numberRequests := 0
		for priority := range proxies {
			for _, proxyURL := range proxies[priority] {
				go CheckProxy(proxyURL, priority, ch)
				numberRequests++
			}
		}

		// getting results
		for i := 1; i <= numberRequests; i++ {
			checkResult := <-ch
			proxyURL := checkResult.ProxyURL
			results[proxyURL] = checkResult
		}

		healthcheckSuccessful := true
		for proxyURL, checkResult := range results {
			if checkResult.Success == false {
				reportProxyUnhealthy(proxyURL)
				pp.MarkProxyAsUnhealthy(checkResult.ProxyPriority, proxyURL)
				healthcheckSuccessful = false
			} else {
				pp.MarkProxyAsHealthy(checkResult.ProxyPriority, proxyURL)
			}
		}

		if healthcheckSuccessful {
			log.Printf("Proxies healthcheck successful")
		}

		time.Sleep(healthcheckInterval)
	}
}

func reportProxyUnhealthy(proxyURL string) {
	msg := fmt.Sprintf("Proxy %s is unhealthy", proxyURL)
	log.Println(msg)
	promNotifier := sources.GetPrometheusNotifierInstance()
	promNotifier.Notify(msg, "proxyPoolService")
}
