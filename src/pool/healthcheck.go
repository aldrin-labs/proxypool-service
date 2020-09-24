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
	Success    bool                 `json:"success"`
	UsedWeight string               `json:"usedWeight"`
	Response   *BinancePingResponse `json:"response"`
}

func CheckProxy(proxyURL string) HealthCheckResponse {
	binanceFapiEndpoint := "https://fapi.binance.com/fapi/v1/time"
	// binanceSpotEndpoint := "https://api.binance.com/api/v3/exchangeInfo"

	rawResult, headers := MakeHTTPRequestUsingProxy(binanceFapiEndpoint, proxyURL)

	// log.Printf("%v", rawResult)
	usedWeight := headers.Get("X-MBX-USED-WEIGHT-1m")

	result := BinancePingResponse{}
	hcResponse := HealthCheckResponse{
		Success:    false,
		UsedWeight: usedWeight,
		Response:   &result,
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
