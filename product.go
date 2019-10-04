package main

import (
	"fmt"
	"time"
    "encoding/json"
)

// Product Sub-Service
type productItem struct {
    PID     string
    Score   float64
    Name    string
    Url     string
    Price   float64
    Image   string
}
type productResult []productItem

func parseProductResult(body []byte) (productResult, error) {
    var raw []interface{}
    if err := json.Unmarshal(body, &raw); err != nil {
        return nil, err
    }
    var products productResult
    for _, m := range raw {
        n := m.([]interface{})
        var product productItem
        product.PID = n[0].(string)
        product.Score = n[1].(float64)
        products = append(products, product)
    }
    return products, nil
}

func handleProductFake(bodyBytes []byte, pChanOut chan productResult) (productResult, error) {
    p1 := productItem {
        PID : "JD-30034871415",
        Score: 0.228515625,
    }
    p2 := productItem {
        PID : "JD-39688051397",
        Score: 0.2161865234375,
    }
    p3 := productItem {
        PID : "JD-39378786216",
        Score: 0.2161865234375,
    }
    products := productResult{p1, p2, p3}
    pChanOut <- products
    return products, nil
}

func handleProduct(bodyBytes []byte, pChanOut chan productResult) (productResult, error) {
    // bodyBytes, _ := ioutil.ReadAll(c.Request.Body)
    t1 := time.Now()
    body, err := redirectToSubService(productUrl, bodyBytes)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return nil, err
    }
    t2 := time.Since(t1).Seconds()
    fmt.Printf("HandleProduct Time Elapsed: %fs\n", t2)
    result, err := parseProductResult(body)
    // fmt.Println(result)
    // c.String(http.StatusOK, "success")
    if err == nil {
        pChanOut <- result
    } else {
        pChanOut <- nil
    }
    return result, err
}


