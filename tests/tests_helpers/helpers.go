package tests_helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
)

func MakeHTTPRequest(url string) interface{} {
	req, err := http.NewRequest("GET", url, bytes.NewBuffer([]byte{}))
	if err != nil {
		log.Println("Request error", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Request error", err)
	}

	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	return body
}

func MakeHTTPPostRequest(requrl string, proxy string, counter int) interface{} {
	d := fmt.Sprintf("{\"proxy\":\"%s\",\"counter\":%d}", proxy, counter)
	data := []byte(d)
	r := bytes.NewReader(data)

	req, err := http.NewRequest("POST", requrl, r) //strings.NewReader(data.Encode())
	if err != nil {
		log.Println("Request error", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Request error", err)
	}

	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	return body
}

func MakeProxyRequests(port string, priority int, weight int, numberOfRequests int, wg *sync.WaitGroup) {
	for i := 0; i < numberOfRequests; i++ {
		proxyPoolURL := fmt.Sprintf("http://localhost%s/getProxy?priority=%d&weight=%d", port, priority, weight)
		proxyDataInterface := MakeHTTPRequest(proxyPoolURL)
		proxyDataString := fmt.Sprintf("%s", proxyDataInterface)
		// log.Printf("Got proxy string: %s", proxyDataString)

		proxyData := &struct {
			Proxy   string `json:"proxy"`
			Counter int    `json:"counter"`
		}{}

		json.Unmarshal([]byte(proxyDataString), proxyData)
		// log.Printf("Got proxy data: %v", proxyData.Proxy)

		wg.Done()
	}
}
