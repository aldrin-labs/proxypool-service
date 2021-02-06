package healthcheck

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	loggly_client "gitlab.com/crypto_project/core/proxypool_service/src/sources/loggly"

	"gitlab.com/crypto_project/core/proxypool_service/src/helpers"
	"gitlab.com/crypto_project/core/proxypool_service/src/pool"
	"gitlab.com/crypto_project/core/proxypool_service/src/sources"
)

var healthcheckInterval = 20 * time.Second

func CheckProxy(proxyURL string, proxyHttpClient *http.Client, priority int, ch chan<- HealthCheckResponse) {
	binanceFapiTimeEndpoint := "https://fapi.binance.com/fapi/v1/time"
	binanceSpotEndpoint := "https://api.binance.com/api/v3/exchangeInfo"

	// realIP, country := getProxyInfo(proxyURL)

	hcResponse := HealthCheckResponse{
		Success:       false,
		ProxyURL:      proxyURL,
		ProxyPriority: priority,
	}

	start := time.Now()
	rawResult, futuresHeaders, err := helpers.MakeHTTPRequestUsingProxy(proxyHttpClient, binanceFapiTimeEndpoint)
	if err != nil {
		ch <- hcResponse
		return
	}

	duration := time.Since(start)
	_, spotHeaders, err := helpers.MakeHTTPRequestUsingProxy(proxyHttpClient, binanceSpotEndpoint)
	if err != nil {
		ch <- hcResponse
		return
	}

	usedWeightFutures := futuresHeaders.Get("X-MBX-USED-WEIGHT-1m")
	usedWeightSpot := spotHeaders.Get("X-MBX-USED-WEIGHT-1m")

	result := BinancePingResponse{}
	hcResponse = HealthCheckResponse{
		Success:           false,
		UsedSpotWeight:    usedWeightSpot,
		UsedFuturesWeight: usedWeightFutures,
		ProxyURL:          proxyURL,
		ProxyPriority:     priority,
		// ProxyRealIP:       realIP,
		// ProxyCountry:      country,
		ResponseTimeMs: duration.Milliseconds(),
		Response:       &result,
	}

	jsonErr := json.Unmarshal(rawResult.([]byte), &result)
	if jsonErr != nil {
		loggly_client.GetInstance().Info("Json decode error:", rawResult)
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

func getProxyInfo(proxyHttpClient *http.Client, proxyURL string) (string, string) {
	ipCheckEndpoint := "https://api.myip.com"

	result := IPCheckResponse{}

	rawResult, _, err := helpers.MakeHTTPRequestUsingProxy(proxyHttpClient, ipCheckEndpoint)
	if err != nil {
		return "", ""
	}

	jsonErr := json.Unmarshal(rawResult.([]byte), &result)
	if jsonErr != nil {
		loggly_client.GetInstance().Infof("Json decode error: %s", jsonErr.Error())
	}

	return result.IP, result.Country
}

// warning, this call to binance is not counted in redis rate limiter (but takes only "1" weigth)
// proxyURL format: http://login:pass@ip:port
func RunProxiesHealthcheck() {
	time.Sleep(3 * time.Second)

	pp := pool.GetProxyPoolInstance()
	proxies := pp.Proxies

	// create proxy http clients (one client for one proxy)
	// we will use these clients for whole service lifespan
	proxyHttpClients := CreateProxyHttpClients(proxies)

	for {
		hcStart := time.Now()
		// loggly_client.GetInstance().Infof("Starting proxy healthcheck...")

		results := make(map[string]HealthCheckResponse)

		ch := make(chan HealthCheckResponse)

		numberRequests := 0
		for priority := range proxies {
			for _, proxyURL := range proxies[priority] {

				proxyHttpClient := proxyHttpClients[priority][proxyURL]
				if proxyHttpClient != nil {
					go CheckProxy(proxyURL, proxyHttpClient, priority, ch)
					numberRequests++
				}
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
				pp.GetMetricsClient().Inc("healthcheck.unhealthy_proxy")
			} else {
				pp.MarkProxyAsHealthy(checkResult.ProxyPriority, proxyURL)
			}
		}

		if healthcheckSuccessful {
			duration := time.Since(hcStart)
			loggly_client.GetInstance().Infof("Proxies healthcheck successful: %s", duration)
			pp.GetMetricsClient().Inc("healthcheck.success")
			pp.GetMetricsClient().Timing("healthcheck.duration", int64(duration.Milliseconds()))
		} else {
			pp.GetMetricsClient().Inc("healthcheck.failure")
		}

		time.Sleep(healthcheckInterval)
	}
}

func reportProxyUnhealthy(proxyURL string) {
	msg := fmt.Sprintf("Proxy %s is unhealthy", proxyURL)
	loggly_client.GetInstance().Info(msg)
	promNotifier := sources.GetPrometheusNotifierInstance()
	promNotifier.Notify(msg, "proxyPoolService")
}

// TODO: make this singleton, so we don't create duplicates in healthcheck api call
func CreateProxyHttpClients(proxies [][]string) map[int]map[string]*http.Client {
	proxyHttpClients := map[int]map[string]*http.Client{}

	for priority := range proxies {
		for _, proxyURL := range proxies[priority] {

			parsedProxyURL, err := url.Parse(proxyURL)
			if err != nil {
				loggly_client.GetInstance().Info("ProxyURL parse error", err)
				continue
			}

			proxyClient := &http.Client{
				Transport: &http.Transport{
					Proxy: http.ProxyURL(parsedProxyURL),
					// possible fix for "connection reset by peer"
					MaxConnsPerHost: 50,
				},
				Timeout: 15 * time.Second,
			}

			if proxyHttpClients[priority] == nil {
				proxyHttpClients[priority] = map[string]*http.Client{}
			}

			proxyHttpClients[priority][proxyURL] = proxyClient
		}
	}
	return proxyHttpClients
}
