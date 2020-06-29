package pool

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

type Limit struct {
	requests   int64
	overPeriod int64
}

type Proxy struct {
	counter  int
	locked   bool
	unlockTime time.Time
	mux sync.Mutex
}

type ProxyPool struct {
	Proxies             [][]string
	CurrentProxyIndexes map[int]int
	LimitMap            map[string]*Limit         // Exchange -> Proxy -> Requests Made
	ExchangeProxyMap    map[int]map[string]*Proxy // Exchange -> Proxy -> Requests Made
}

func newProxySingleton() *ProxyPool {
	proxiesBASE64 := os.Getenv("PROXYLIST")
	proxiesJSON, err := base64.StdEncoding.DecodeString(proxiesBASE64)
	if err != nil {
		fmt.Println("error:", err)
		return nil
	}
	var proxies [][]string
	jsonErr := json.Unmarshal([]byte(proxiesJSON), &proxies)
	if jsonErr != nil {
		fmt.Println("json error:", jsonErr)
		return nil
	}

	proxyMap := map[int]map[string]*Proxy{}
	currentProxyIndexes := map[int]int{}

	// 0 - max priority (e.g. for tgrading), 1 - less priority

	proxyMap[0] = map[string]*Proxy{}
	currentProxyIndexes[0] = 0
	proxyMap[1] = map[string]*Proxy{}
	currentProxyIndexes[1] = 0

	for i, proxyArr := range proxies {
		log.Printf("Init %d proxies with %d priority...", len(proxyArr), i)

		for _, proxy := range proxyArr {
			proxyMap[i][proxy] = &Proxy{
				counter: 0, // 240 / min
				locked: false,
				unlockTime: time.Now(),
				mux: sync.Mutex{},
			}
		}
	}

	return &ProxyPool{
		Proxies:             proxies,
		CurrentProxyIndexes: currentProxyIndexes,
		ExchangeProxyMap:    proxyMap,
	}
}

var proxySingleton *ProxyPool

func GetProxyPoolInstance() *ProxyPool {
	if proxySingleton == nil {
		proxySingleton = newProxySingleton()
	}
	return proxySingleton
}

func (pp *ProxyPool) GetProxyByPriority(priority int) string {
	if pp.Proxies == nil {
		return ""
	}
	log.Printf("Got GetProxyByPriority request with %d priority", priority)

	// check for unlocked high-priority proxies
	existHighPriorityUnlockedProxy := false
	for _, proxy := range pp.ExchangeProxyMap[0] {
		if !proxy.locked || proxy.locked && proxy.unlockTime.Before(time.Now()) {
			existHighPriorityUnlockedProxy = true
		}
	}

	if !existHighPriorityUnlockedProxy {
		priority = 1
	}

	// check for unlocked low-priority proxies
	existLowPriorityUnlockedProxy := false
	for _, proxy := range pp.ExchangeProxyMap[1] {
		if !proxy.locked || (proxy.locked && proxy.unlockTime.Before(time.Now())) {
			existLowPriorityUnlockedProxy = true
		}
	}

	if !existLowPriorityUnlockedProxy {
		time.Sleep(1 * time.Minute)
	}

	currentIndex := pp.CurrentProxyIndexes[priority]
	pp.CurrentProxyIndexes[priority] = currentIndex + 1
	if currentIndex >= len(pp.Proxies[priority]) {
		pp.CurrentProxyIndexes[priority] = 1
		currentIndex = 0
	}

	currentProxy := pp.Proxies[priority][currentIndex]
	currentRequests := pp.ExchangeProxyMap[priority][currentProxy]

	currentRequests.mux.Lock()
	defer currentRequests.mux.Unlock()

	if currentRequests.counter >= 240 && currentRequests.locked && currentRequests.unlockTime.After(time.Now()) {
		return pp.GetProxyByPriority(priority)
	} else if currentRequests.counter >= 240 && !currentRequests.locked {
		// we block proxy if more than 240 requests per minute were executed
		currentRequests.locked = true
		currentRequests.unlockTime = time.Now().Add(1 * time.Minute)
		return pp.GetProxyByPriority(priority)
	} else {
		currentRequests.counter += 1
		return currentProxy
	}
}
