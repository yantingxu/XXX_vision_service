package main

import (
	"fmt"
	"time"
    "strings"
    "strconv"
    "net/http"
    "io/ioutil"
    "encoding/json"
)

// Price Sub-Service
type priceItem struct {
    Id      string     `json:"id"`
    RawPrice   string     `json:"p"`
    Price   float64
}
type priceResult []priceItem

func parsePriceResult(body []byte) (priceResult, error) {
    var result priceResult
    if err := json.Unmarshal(body, &result); err != nil {
        fmt.Println(err)
        return nil, err
    }
    for i, price := range result {
        result[i].Id = strings.TrimPrefix(price.Id, "J_")
        result[i].Price, _ = strconv.ParseFloat(price.RawPrice, 64)
        // fmt.Println(result[i])
    }
    return result, nil
}

func handlePriceFake(products productResult, chanOut chan priceResult) (priceResult, error) {
    p1 := priceItem {
        Id : "30034871415",
        Price : 8.9,
    }
    p2 := priceItem {
        Id : "39688051397",
        Price : 11.5,
    }
    p3 := priceItem {
        Id : "39378786216",
        Price : 11.5,
    }
    result := priceResult{p1, p2, p3}
    chanOut <- result
    return result, nil
}

func handlePrice(products productResult, chanOut chan priceResult) (priceResult, error) {
    var pids []string
    for _, p := range products {
        if strings.HasPrefix(p.PID, "JD-") {
            pids = append(pids, strings.TrimPrefix(p.PID, "JD-"))
        }
    }
    req, err := http.NewRequest("GET", priceUrl, nil)
    if err != nil {
        return nil, err
    }
    q := req.URL.Query()
    q.Add("skuids", strings.Join(pids, ","))
    req.URL.RawQuery = q.Encode()
    fmt.Println(req.URL.String())

    t1 := time.Now()
    client := http.Client{}
    resp, err := client.Do(req)
    t2 := time.Since(t1).Seconds()
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    fmt.Printf("HandlePrice Time Elapsed: %fs\n", t2)

    body, _ := ioutil.ReadAll(resp.Body)
    result, err := parsePriceResult(body)
    if err != nil {
        return nil, err
    }
    if err == nil {
        chanOut <- result
    } else {
        chanOut <- nil
    }
    // fmt.Println(result)
    return result, nil
}

