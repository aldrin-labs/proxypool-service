package helpers

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"

	loggly_client "gitlab.com/crypto_project/core/proxypool_service/src/sources/loggly"
)

type HTTPResponseStruct struct {
	Body    interface{}
	Headers http.Header
}

func GetProxiesFromENV(proxies *[][]string) {
	proxiesBASE64 := os.Getenv("PROXYLIST")
	loggly_client.GetInstance().Info("proxiesBASE64 ", proxiesBASE64)
	proxiesJSON, err := base64.StdEncoding.DecodeString(string(proxiesBASE64))
	if err != nil {
		loggly_client.GetInstance().Info("error:", err)
		return
	}
	jsonErr := json.Unmarshal([]byte(proxiesJSON), proxies)
	if jsonErr != nil {
		loggly_client.GetInstance().Info("json error:", jsonErr)
		return
	}
	// Uncomment this to test with only one proxy for each priority
	for idx, row := range *proxies {
		(*proxies)[idx] = []string{row[0]}
	}
	loggly_client.GetInstance().Info("Testing with only one proxy per priority: ", *proxies)
}

func FindIP(input string) string {
	numBlock := "(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])"
	regexPattern := numBlock + "\\." + numBlock + "\\." + numBlock + "\\." + numBlock
	regEx := regexp.MustCompile(regexPattern)
	return regEx.FindString(input)
}

// MakeHTTPRequestUsingProxy
func MakeHTTPRequestUsingProxy(proxyHttpClient *http.Client, URL string) (interface{}, http.Header, error) {

	var body []byte
	var headers http.Header

	resp, err := proxyHttpClient.Get(URL)
	if err != nil {
		loggly_client.GetInstance().Info("[MakeHTTPRequestUsingProxy] Request error: ", err)
		return body, headers, err
	}

	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		loggly_client.GetInstance().Info("[MakeHTTPRequestUsingProxy] Request body read error: ", err)
		return body, headers, err
	}

	return body, resp.Header, nil
}
