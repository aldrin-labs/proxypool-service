package tests

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

func makeHTTPPostRequest(requrl string, proxy string, counter int) interface{} {
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
