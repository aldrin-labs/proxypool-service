package pool

import (
	"encoding/json"
	"log"
	"time"
)

type BinancePingResponse struct {
	ServerTime int    `json:"serverTime"`
	Code       string `json:"code,omitempty"`
	Msg        string `json:"msg,omitempty"`
}

type HealthCheckResponse struct {
	Success           bool                 `json:"success"`
	UsedSpotWeight    string               `json:"usedSpotWeight"`
	UsedFuturesWeight string               `json:"usedFuturesWeight"`
	ProxyURL          string               `json:"proxyURL"`
	ProxyPriority     int                  `json:"proxyPriority"`
	ProxyCountry      string               `json:"proxyCountry"`
	ProxyRealIP       string               `json:"proxyRealIp"`
	ResponseTimeMs    int64                `json:"responseTimeMs"`
	Response          *BinancePingResponse `json:"response"`
}

type IPCheckResponse struct {
	IP      string `json:"ip"`
	Country string `json:"country"`
	CC      string `json:"cc"`
}

func CheckProxy(proxyURL string, priority int, ch chan<- HealthCheckResponse) {
	binanceFapiTimeEndpoint := "https://fapi.binance.com/fapi/v1/time"
	binanceSpotEndpoint := "https://api.binance.com/api/v3/exchangeInfo"

	realIP, country := getProxyInfo(proxyURL)

	start := time.Now()
	rawResult, futuresHeaders := MakeHTTPRequestUsingProxy(binanceFapiTimeEndpoint, proxyURL)
	duration := time.Since(start)
	_, spotHeaders := MakeHTTPRequestUsingProxy(binanceSpotEndpoint, proxyURL)

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

	rawResult, _ := MakeHTTPRequestUsingProxy(ipCheckEndpoint, proxyURL)

	jsonErr := json.Unmarshal(rawResult.([]byte), &result)
	if jsonErr != nil {
		log.Printf("Json decode error: %s", jsonErr.Error())
	}

	return result.IP, result.Country
}
