package pool

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"log"

	"golang.org/x/time/rate"
)

type Proxy struct {
	RateLimiter   *rate.Limiter
	Usages        int
	Name          string
	Locked        bool
	Limit         float64
	NeedResponses int
}

type ProxyResponse struct {
	Proxy   string `json:"proxy"`
	Counter int    `json:"counter"`
}

type ProxyPool struct {
	Proxies             [][]string
	CurrentProxyIndexes map[int]int
	ExchangeProxyMap    map[int]map[string]*Proxy
	DebtorsMap          map[string]time.Time

	StartupTime     time.Time
	Timeout         int
	proxyIndexesMux sync.Mutex
	proxyStatsMux   sync.Mutex
}

var proxySingleton *ProxyPool

func newProxySingleton() *ProxyPool {
	var proxies [][]string
	getProxiesFromENV(&proxies)

	proxyMap := map[int]map[string]*Proxy{}
	currentProxyIndexes := map[int]int{}

	// 0 - max priority (e.g. for trading), 1 - less priority
	proxyMap[0] = map[string]*Proxy{}
	currentProxyIndexes[0] = 0
	proxyMap[1] = map[string]*Proxy{}
	currentProxyIndexes[1] = 0

	normalLimit := 0.33                           // 60 / min
	normalRateLimit := rate.Limit(normalLimit) // 60 / min
	// how much requests can be run simultaneously if there were no throttling when they were received
	burst := 1

	for i, proxyArr := range proxies {
		log.Printf("Init %d proxies with %d priority...", len(proxyArr), i)

		for _, proxy := range proxyArr {
			proxyMap[i][proxy] = &Proxy{
				RateLimiter:   rate.NewLimiter(normalRateLimit, burst),
				Usages:        0,
				Limit:         normalLimit,
				Locked:        false,
				NeedResponses: 0,
				Name:          proxy,
			}
		}
	}

	return &ProxyPool{
		Proxies:             proxies,
		CurrentProxyIndexes: currentProxyIndexes,
		ExchangeProxyMap:    proxyMap,
		DebtorsMap:          map[string]time.Time{},
		proxyIndexesMux:     sync.Mutex{},
		proxyStatsMux:       sync.Mutex{},
		StartupTime:         time.Now(),
		Timeout:             90,
	}
}

func GetProxyPoolInstance() *ProxyPool {
	if proxySingleton == nil {
		proxySingleton = newProxySingleton()
		go proxySingleton.CheckProxyTimeout()
	}
	return proxySingleton
}

func getProxiesFromENV(proxies *[][]string) {
	proxiesBASE64 := os.Getenv("PROXYLIST")
	log.Println("proxiesBASE64 ", proxiesBASE64)
	proxiesJSON, err := base64.StdEncoding.DecodeString(string(proxiesBASE64))
	log.Print("proxiesJSON ", proxiesJSON)
	if err != nil {
		log.Print("error:", err)
		return
	}
	jsonErr := json.Unmarshal([]byte(proxiesJSON), proxies)
	if jsonErr != nil {
		log.Print("json error:", jsonErr)
		return
	}
}

func (pp *ProxyPool) GetProxyByPriority(priority int) ProxyResponse {
	if pp.Proxies == nil {
		return ProxyResponse{Proxy: "", Counter: 0}
	}

	// TODO: maybe it's better to use sync.map here
	pp.proxyIndexesMux.Lock()
	currentIndex := pp.CurrentProxyIndexes[priority]
	pp.CurrentProxyIndexes[priority] = currentIndex + 1
	if currentIndex >= len(pp.Proxies[priority]) {
		pp.CurrentProxyIndexes[priority] = 1
		currentIndex = 0
	}

	currentProxyURL := pp.Proxies[priority][currentIndex]
	currentProxy := pp.ExchangeProxyMap[priority][currentProxyURL]
	currentProxyRateLimiter := currentProxy.RateLimiter
	pp.proxyIndexesMux.Unlock()

	if float64(currentProxy.NeedResponses) >= currentProxy.Limit {
		if priority == 0 {
			log.Print("Top priority proxy is blocked. Returning low priority proxy.")
			return pp.GetLowPriorityProxy()
		}

		ctx := context.Background()
		err := currentProxyRateLimiter.Wait(ctx)
		if err != nil {
			log.Print("Error proxy wait", err.Error())
		}
		return pp.GetTopPriorityProxy()
	}

	pp.proxyStatsMux.Lock()
	currentProxy.Usages++
	currentProxy.NeedResponses++
	pp.DebtorsMap[currentProxyURL+"_"+strconv.Itoa(currentProxy.Usages)] = time.Now()
	pp.proxyStatsMux.Unlock()

	log.Print("return proxy url: ", currentProxyURL, " proxy, needResponses: ", currentProxy.NeedResponses)
	return ProxyResponse{
		Proxy:   currentProxyURL,
		Counter: currentProxy.Usages,
	}
}

func (pp *ProxyPool) ExemptProxy(url string, counter int) {
	println("exempt url counter", url, counter)
	pp.proxyStatsMux.Lock()
	for priority, proxyArr := range pp.Proxies {
		for _, proxy := range proxyArr {
			if proxy == url && pp.ExchangeProxyMap[priority][proxy].NeedResponses > 0 {
				pp.ExchangeProxyMap[priority][proxy].NeedResponses--
				pp.DebtorsMap[proxy+"_"+strconv.Itoa(counter)] = time.Time{}
				log.Print("ExemptProxy url: ", url, "new needResponses: ", pp.ExchangeProxyMap[priority][proxy].NeedResponses)
			}
		}
	}
	pp.proxyStatsMux.Unlock()
}

func (pp *ProxyPool) GetLowPriorityProxy() ProxyResponse {
	return pp.GetProxyByPriority(1)
}

func (pp *ProxyPool) GetTopPriorityProxy() ProxyResponse {
	return pp.GetProxyByPriority(0)
}

func (pp *ProxyPool) GetStats() []string {
	stats := []string{}
	timeSinceStartup := time.Since(pp.StartupTime).Seconds()

	for priority := range pp.ExchangeProxyMap {
		for _, proxy := range pp.ExchangeProxyMap[priority] {
			proxyIP := findIP(proxy.Name)
			data := fmt.Sprintf("Proxy %s with priority %d got %f requests/sec on avg \n", proxyIP, priority, float64(proxy.Usages)/timeSinceStartup)
			stats = append(stats, data)
		}
	}

	return stats
}

func (pp *ProxyPool) CheckProxyTimeout() {
	// timeout func here
	for {
		time.Sleep(30 * time.Second)
		for k, v := range pp.DebtorsMap {
			if time.Since(v).Seconds() >= float64(pp.Timeout) && !v.IsZero() {
				arr := strings.Split(k, "_")
				counter, _ := strconv.Atoi(arr[1])
				pp.ExemptProxy(arr[0], counter)
			}
		}
	}
}

func findIP(input string) string {
	numBlock := "(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])"
	regexPattern := numBlock + "\\." + numBlock + "\\." + numBlock + "\\." + numBlock
	regEx := regexp.MustCompile(regexPattern)
	return regEx.FindString(input)
}
