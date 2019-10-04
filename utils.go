package main

import (
    "bytes"
    "net/http"
    "io/ioutil"
)

var client *http.Client

func init() {
    client = &http.Client {
        Timeout: clientTimeout,
    }
}

func redirectToSubService(url string, bodyBytes []byte) ([]byte, error) {
    bodyReader := bytes.NewReader(bodyBytes)
    request, err := http.NewRequest("POST", url, bodyReader)
    if err != nil {
        return nil, err
    }
    request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    resp, err := client.Do(request)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // statusCode, contentLength, contentType := resp.StatusCode, resp.ContentLength, resp.Header.Get("Content-Type")
    // fmt.Println(contentType)
    return ioutil.ReadAll(resp.Body)
}


