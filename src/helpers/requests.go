package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	loggly_client "gitlab.com/crypto_project/core/proxypool_service/src/sources/loggly"
	"io/ioutil"
	"net/http"
)

func MakeGetRequest(baseURL string, apiMethod string, params map[string]string) (interface{}, error) {
	URL := baseURL + "/" + apiMethod

	loggly_client.GetInstance().Infof("[HTTP] Making GET request to %s...", URL)

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

	loggly_client.GetInstance().Infof("[HTTP] Making POST request to %s...", URL)

	jsonStr, err := json.Marshal(data)
	if err != nil {
		loggly_client.GetInstance().Infof("[HTTP] Error while encoding JSON in POST request: %s", err.Error())
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
