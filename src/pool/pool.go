package pool

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Limit struct {
	requests int64
	overPeriod int64
}


type Proxy struct {
	RequestsMade int64
	LastTimestamp int64
}

type ProxyPool struct{
	Proxies [] string
	CurrentProxyIndex int
	LimitMap map[string] *Limit // Exchange -> Proxy -> Requests Made
	ExchangeProxyMap map[string]map[string] *Proxy // Exchange -> Proxy -> Requests Made
}

// NewSignalSingleton returns SignalSingleton instance
func newProxySingleton() *ProxyPool {
	proxiesBASE64 := os.Getenv("PROXYLIST")
	proxiesJSON, err := base64.StdEncoding.DecodeString(proxiesBASE64)
	if err != nil {
		fmt.Println("error:", err)
		return nil
	}
	var proxies [] string

	json.Unmarshal([]byte(proxiesJSON), &proxies)

	// exchanges := [2]string{"binance", "bittrex"}

	proxyMap := map[string]map[string]*Proxy{}

	proxyMap["binance"] = map[string]*Proxy{}
	proxyMap["bittrex"] = map[string]*Proxy{}

	for _, proxy := range proxies {
		proxyMap["binance"][proxy] = &Proxy{RequestsMade: 0, LastTimestamp: time.Now().UnixNano()}
		proxyMap["bittrex"][proxy] = &Proxy{RequestsMade: 0, LastTimestamp: time.Now().UnixNano()}
	}

	limitMap := map[string] *Limit{}
	limitMap["binance"] = &Limit{
		requests: 1,
		overPeriod: 800,
	}
	limitMap["bittrex"] = &Limit{
		requests: 1,
		overPeriod: 1200,
	}

	// env PROXYLIST
	return &ProxyPool{
		Proxies:proxies,
		CurrentProxyIndex: 0,
		ExchangeProxyMap: proxyMap,
		LimitMap: limitMap,
	}
}

var proxySingleton *ProxyPool

func GetProxyPoolInstance() *ProxyPool {
	if proxySingleton == nil {
		proxySingleton = newProxySingleton()
	}
	return proxySingleton
}


func (pp *ProxyPool) GetProxyByExchange(exchangeName string) string {
	currentIndex := pp.CurrentProxyIndex
	pp.CurrentProxyIndex = pp.CurrentProxyIndex + 1
	if pp.CurrentProxyIndex >= len(pp.Proxies) {
		pp.CurrentProxyIndex = 0
	}
	currentProxy := pp.Proxies[currentIndex]

	currentTime := time.Now().UnixNano()
	currentRequests := pp.ExchangeProxyMap[exchangeName][currentProxy]
	limit := pp.LimitMap[exchangeName]
	pp.ExchangeProxyMap[exchangeName][currentProxy].RequestsMade += 1
	// if made more requests than in limit faster than given period
	if currentRequests.RequestsMade > limit.requests &&
		currentTime - currentRequests.LastTimestamp < limit.overPeriod {
		duration :=
			time.Millisecond * time.Duration(limit.overPeriod - (currentTime - currentRequests.LastTimestamp))

		time.Sleep(duration)
	}

	currentTime = time.Now().UnixNano()
	if currentTime - currentRequests.LastTimestamp > limit.overPeriod {
		pp.ExchangeProxyMap[exchangeName][currentProxy].LastTimestamp = currentTime
	}

	return currentProxy
}

