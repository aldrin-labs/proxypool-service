package pool

import (
	"encoding/json"
	"log"
)

type BinancePingResponse struct {
	ServerTime int    `json:"serverTime"`
	Code       string `json:"code,omitempty"`
	Msg        string `json:"msg,omitempty"`
}

type HealthCheckResponse struct {
	Success       bool                 `json:"success"`
	UsedWeight    string               `json:"usedWeight"`
	ProxyPriority int                  `json:"proxyPriority"`
	ProxyCountry  string               `json:"proxyCountry"`
	ProxyRealIP   string               `json:"proxyRealIp"`
	Response      *BinancePingResponse `json:"response"`
}

type IPCheckResponse struct {
	IP      string `json:"ip"`
	Country string `json:"country"`
	CC      string `json:"cc"`
}

func CheckProxy(proxyURL string, priority int) HealthCheckResponse {
	binanceFapiEndpoint := "https://fapi.binance.com/fapi/v1/time"
	// binanceSpotEndpoint := "https://api.binance.com/api/v3/exchangeInfo"

	realIP, country := getProxyInfo(proxyURL)

	rawResult, headers := MakeHTTPRequestUsingProxy(binanceFapiEndpoint, proxyURL)

	usedWeight := headers.Get("X-MBX-USED-WEIGHT-1m")

	result := BinancePingResponse{}
	hcResponse := HealthCheckResponse{
		Success:       false,
		UsedWeight:    usedWeight,
		ProxyPriority: priority,
		ProxyRealIP:   realIP,
		ProxyCountry:  country,
		Response:      &result,
	}

	jsonErr := json.Unmarshal(rawResult.([]byte), &result)
	if jsonErr != nil {
		log.Print("Json decode error:", rawResult)
		return hcResponse
	}

	if result.ServerTime > 0 {
		hcResponse.Success = true
		return hcResponse
	}

	return hcResponse
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
