package main

import (
	"fmt"
	"time"
    "net/url"
    "strconv"
    "encoding/json"
)

// Rerank Sub-Service
type rerankResult []string
type rerankCand [2]string
type rerankAttrs struct {
    Fashion []float64       `json:"fashion"`
    Logo    []string        `json:"logo"`
}
type rerankInput struct {
    Candidates  []rerankCand    `json:"candidates"`
    Attrs       rerankAttrs     `json:"attrs"`
    ClassId     int             `json:"classid"`
    RandomValue string          `json:"randomValue"`
}

func parseRerankResult(body []byte) (rerankResult, error) {
    var result rerankResult
    if err := json.Unmarshal(body, &result); err != nil {
        return nil, err
    }
    return result, nil
}

func handleRerankFake(products productResult, fashion fashionResult, classid int, randomValue string, chanOut chan rerankResult) (rerankResult, error) {
    result := rerankResult{"JD-30034871415", "JD-39688051397", "JD-39378786216"}
    chanOut <- result
    return result, nil
}

func handleRerank(products productResult, fashion fashionResult, classid int, randomValue string, chanOut chan rerankResult) (rerankResult, error) {
    var candidates []rerankCand
    for _, p := range products {
        cand := rerankCand {
            p.PID,
            strconv.FormatFloat(p.Score, 'f', 6, 64),
        }
        candidates = append(candidates, cand)
    }
    candidates_str, _ := json.Marshal(candidates)
    attrs := rerankAttrs {
        Fashion : []float64(fashion),
        Logo : []string{},
    }
    attrs_str, _ := json.Marshal(attrs)
    // randomValue := "12321"
    param := url.Values {
        "candidates": {string(candidates_str)},
        "attrs": {string(attrs_str)},
        "classid": {strconv.FormatInt(int64(classid), 10)},
        "randomValue": {randomValue},
    }
    // fmt.Println(param.Encode())

    t1 := time.Now()
    body, err := redirectToSubService(rerankUrl, []byte(param.Encode()))
    if err != nil {
        return nil, err
    }
    t2 := time.Since(t1).Seconds()
    fmt.Printf("HandleRerank Time Elapsed: %fs\n", t2)

    // fmt.Println(string(body))
    result, err := parseRerankResult(body)
    if err == nil {
        chanOut <- result
    } else {
        chanOut <- nil
    }
    return result, err
}

