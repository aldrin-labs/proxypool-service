package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

func MakeGetRequest(baseURL string, apiMethod string, params map[string]string) (interface{}, error) {
	URL := baseURL + "/" + apiMethod

	log.Printf("[HTTP] Making GET request to %s...", URL)

	if len(params) > 0 {
		URL = URL + "?"
		for k, v := range params {
			URL = URL + fmt.Sprintf("%s=%s&", k, v)
		}
		// trim last &
		last := len(URL) - 1
		URL = URL[:last]
	}

	response, err := makeHTTPRequest(URL, "GET", bytes.NewBuffer([]byte{}))

	return response, err
}

func MakePostRequest(baseURL string, apiMethod string, data interface{}) (interface{}, error) {
	URL := baseURL + "/" + apiMethod

	log.Printf("[HTTP] Making POST request to %s...", URL)

	jsonStr, err := json.Marshal(data)
	if err != nil {
		log.Printf("[HTTP] Error while encoding JSON in POST request: %s", err.Error())
		return nil, err
	}

	response, err := makeHTTPRequest(URL, "POST", bytes.NewBuffer(jsonStr))

	return response, err
}

func makeHTTPRequest(url string, requestType string, data *bytes.Buffer) (interface{}, error) {

	var resp *http.Response
	var err error
	if requestType == "POST" {
		resp, err = http.Post(url, "application/json", data)
	} else {
		resp, err = http.Get(url)
	}

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	var response interface{}
	err = json.Unmarshal(body, &response)

	if err != nil {
		return nil, err
	}

	return response, nil
}
