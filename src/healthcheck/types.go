package healthcheck

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
