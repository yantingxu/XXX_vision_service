package main

import (
	"fmt"
	"time"
    "encoding/json"
)

// Fashion Sub-Service
type fashionResult []float64

func parseFashionResult(body []byte) (fashionResult, error) {
    var result fashionResult
    if err := json.Unmarshal(body, &result); err != nil {
        return nil, err
    }
    return result, nil
}

func handleFashionFake(bodyBytes []byte, fChanOut chan *fashionResult) (fashionResult, error) {
    result := fashionResult{0.0, 0.0, 0.0}
    fChanOut <- &result
    return result, nil
}

func handleFashion(bodyBytes []byte, fChanOut chan fashionResult) (fashionResult, error) {
    // bodyBytes, _ := ioutil.ReadAll(c.Request.Body)
    t1 := time.Now()
    body, err := redirectToSubService(fashionUrl, bodyBytes)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return nil, err
    }
    t2 := time.Since(t1).Seconds()
    fmt.Printf("HandleFashion Time Elapsed: %fs\n", t2)

    result, err := parseFashionResult(body)
    // fmt.Println(result)
    if err == nil {
        fChanOut <- result
    } else {
        fChanOut <- nil
    }
    return result, err
}





