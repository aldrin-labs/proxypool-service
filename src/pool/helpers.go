package pool

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

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

func findIP(input string) string {
	numBlock := "(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])"
	regexPattern := numBlock + "\\." + numBlock + "\\." + numBlock + "\\." + numBlock
	regEx := regexp.MustCompile(regexPattern)
	return regEx.FindString(input)
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
