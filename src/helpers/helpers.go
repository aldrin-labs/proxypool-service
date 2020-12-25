package helpers

import (
	"encoding/base64"
	"encoding/json"
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

func GetProxiesFromENV(proxies *[][]string) {
	proxiesBASE64 := os.Getenv("PROXYLIST")
	log.Println("proxiesBASE64 ", proxiesBASE64)
	proxiesJSON, err := base64.StdEncoding.DecodeString(string(proxiesBASE64))
	// log.Print("proxiesJSON ", proxiesJSON)
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

func FindIP(input string) string {
	numBlock := "(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])"
	regexPattern := numBlock + "\\." + numBlock + "\\." + numBlock + "\\." + numBlock
	regEx := regexp.MustCompile(regexPattern)
	return regEx.FindString(input)
}

// MakeHTTPRequestUsingProxy - proxyURL format: http://login:pass@ip:port
func MakeHTTPRequestUsingProxy(URL string, proxyURL string) (interface{}, http.Header, error) {

	parsedProxyURL, err := url.Parse(proxyURL)
	if err != nil {
		log.Println("ProxyURL parse error", err)
	}

	myClient := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(parsedProxyURL)},
		Timeout:   15 * time.Second,
	}

	var body []byte
	var headers http.Header

	resp, err := myClient.Get(URL)
	if err != nil {
		log.Println("Request error", err)
		return body, headers, err
	}

	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Request error", err)
		return body, headers, err
	}

	return body, resp.Header, nil
}
