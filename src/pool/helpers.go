package pool

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"
)

type HTTPResponseStruct struct {
	Body    interface{}
	Headers http.Header
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

func (pp *ProxyPool) GetStats() []string {
	stats := []string{}
	timeSinceStartup := time.Since(pp.StartupTime).Seconds()

	for priority := range pp.ExchangeProxyMap {
		for _, proxy := range pp.ExchangeProxyMap[priority] {
			proxyIP := findIP(proxy.URL)
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

// MakeHTTPRequestUsingProxy - proxyURL format: http://login:pass@ip:port
func MakeHTTPRequestUsingProxy(URL string, proxyURL string) (interface{}, http.Header) {

	parsedProxyURL, err := url.Parse(proxyURL)
	if err != nil {
		log.Println("ProxyURL parse error", err)
	}

	myClient := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(parsedProxyURL)}}

	var body []byte
	var headers http.Header

	resp, err := myClient.Get(URL)
	if err != nil {
		log.Println("Request error", err)
		return body, headers
	}

	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Request error", err)
		return body, headers
	}

	return body, resp.Header
}
