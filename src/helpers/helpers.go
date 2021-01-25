package helpers

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	// "os"
	"regexp"
	"time"
)

type HTTPResponseStruct struct {
	Body    interface{}
	Headers http.Header
}

func GetProxiesFromENV(proxies *[][]string) {
	proxiesBASE64 := "W1siaHR0cDovL05rYkZnVzpVa0pRQU9mUTNiQDEwOS4yNDguNDguMTExOjEwNTAiXSxbImh0dHA6Ly9Oa2JGZ1c6VWtKUUFPZlEzYkAxOTMuNTguMTY4LjEwOjEwNTAiLCJodHRwOi8vTmtiRmdXOlVrSlFBT2ZRM2JAMTg4LjEzMC4yMTguMTk5OjEwNTAiLCJodHRwOi8vTmtiRmdXOlVrSlFBT2ZRM2JAMTk0LjMyLjIzNy4xNzY6MTA1MCIsImh0dHA6Ly9Oa2JGZ1c6VWtKUUFPZlEzYkA5NS4xODIuMTI3LjIzOjEwNTAiLCJodHRwOi8vTmtiRmdXOlVrSlFBT2ZRM2JAMTA5LjI0OC4yMDQuMTY3OjEwNTAiLCJodHRwOi8vTmtiRmdXOlVrSlFBT2ZRM2JAOTQuMTU4LjE5MC41OjEwNTAiLCJodHRwOi8vTmtiRmdXOlVrSlFBT2ZRM2JANDUuMTM5LjE3Ni4yNTM6MTA1MCIsImh0dHA6Ly9Oa2JGZ1c6VWtKUUFPZlEzYkA0NS4xNDIuMjUzLjEwMzoxMDUwIiwiaHR0cDovL05rYkZnVzpVa0pRQU9mUTNiQDQ1LjEzNC4xODEuNzI6MTA1MCJdXQ=="
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
		Timeout:   10 * time.Second,
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
