package proxypool

 type ProxyPool struct{
	proxies [] string
	currentProxyIndex int
	ExchangeProxyMap map[string]map[string] int // Exchange -> Proxy -> Requests Made
}

// NewSignalSingleton returns SignalSingleton instance
func newProxySingleton() *ProxyPool {
	// env PROXYLIST
	return &ProxyPool{
	}
}

var proxySingleton *ProxyPool

func GetProxyPool() *ProxyPool {
	if proxySingleton == nil {
		proxySingleton = newProxySingleton()
	}
	return proxySingleton
}


func (pp *ProxyPool) GetProxyByExchange(exchangeName string) string {
	return "test"
}

