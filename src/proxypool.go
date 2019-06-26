package main

 type ProxyPool struct{
	proxies [] string
	currentProxyIndex int
	ExchangeProxyMap map[string]map[string] int // Exchange -> Proxy -> Requests Made
}

