package main

import (
    // "bytes"
    "flag"
    "strings"
	"fmt"
    "encoding/base64"
    // "encoding/json"
    "net/http"
    "net/url"
    "io/ioutil"
    "time"

	"gocv.io/x/gocv"
)

// const URL = "http://192.168.1.11:8000/fashion"
// const URL = "http://192.168.1.11:8000/product"
// const URL = "http://localhost:9070/product"
// const URL = "http://localhost:9070/shopping"
// const URL = "http://192.168.1.11:8889/shopping"

var (
    URL         string
)

func init() {
    flag.StringVar(&URL, "url", "http://localhost:9070/shopping", "url")
}

// product Handler Param
type inputParam struct {
    Code string  `form:"code"`
    Content     string  `form:"content"`
    DbId        int `form:"db_id"`
    ClassId        int `form:"classid"`
    RandomValue string  `form:"randomValue"`
}

func main() {
    flag.Parse()

    // URL = fmt.Sprintf("http://localhost:%s/shopping", httpPort)

    filename := "2019-01-14 18-29-31.jpg"
    img := gocv.IMRead(filename, gocv.IMReadColor)
    fmt.Println(filename, img.Size())
    bs, _ := gocv.IMEncode(gocv.JPEGFileExt, img)
    encStr := base64.StdEncoding.EncodeToString(bs)
    // fmt.Println(encStr)

    param := url.Values {
        "code": {"100100"},
        "content": {encStr},
        "randomValue": {"123"},
        "db_id": {"101"},
        "classid": {"0"},
    }
    request, err := http.NewRequest("POST", URL, strings.NewReader(param.Encode()))
    if err != nil {
        fmt.Println(err)
        return
    }
    request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    t1 := time.Now()
    client := http.Client{}
    resp, err := client.Do(request)
    t2 := time.Since(t1).Seconds()
    if err != nil {
        fmt.Println(err)
        return
    }
    defer resp.Body.Close()
    fmt.Printf("Time Elapsed: %fs\n", t2)

    statusCode := resp.StatusCode
    contentType := resp.Header.Get("Content-Type")
    //reader := resp.Body
    body, _ := ioutil.ReadAll(resp.Body)
    fmt.Println(string(body))
    fmt.Println(resp.Header)
    fmt.Println(statusCode)
    fmt.Println(contentType)
}

/*
[["JD-30034871415", 0.228515625], ["JD-39688051397", 0.2161865234375], ["JD-39378786216", 0.2161865234375], ["JD-39937044055", 0.2161865234375], ["JD-39004515974", 0.2161865234375], ["JD-531073", 0.2159423828125], ["JD-1690613", 0.20458984375], ["JD-12295977508", 0.2034912109375], ["JD-39688061505", 0.20263671875], ["JD-39378786215", 0.20263671875], ["JD-39937044063", 0.20263671875], ["JD-39004515973", 0.20263671875], ["JD-39688051399", 0.1978759765625], ["JD-39378786214", 0.1978759765625], ["JD-12295977504", 0.197265625], ["JD-30034882174", 0.192626953125], ["JD-39004515972", 0.18994140625], ["JD-39688051400", 0.1893310546875], ["JD-39004515982", 0.1893310546875], ["JD-39937044058", 0.1893310546875], ["JD-16498581634", 0.188720703125], ["JD-1660126", 0.1883544921875], ["JD-39937044059", 0.1876220703125], ["JD-39688061501", 0.1876220703125], ["JD-39378786218", 0.1876220703125], ["JD-39004515976", 0.1876220703125], ["JD-531072", 0.1871337890625], ["JD-39688051395", 0.1861572265625], ["JD-531070", 0.1861572265625], ["JD-39378786223", 0.1861572265625], ["JD-39937044053", 0.1861572265625], ["JD-39937044061", 0.1849365234375], ["JD-39688061503", 0.1849365234375], ["JD-39378786219", 0.1849365234375], ["JD-39004515977", 0.1849365234375], ["JD-39688050319", 0.1834716796875], ["JD-39378487063", 0.1834716796875], ["JD-39937031808", 0.1834716796875], ["JD-531075", 0.1807861328125], ["JD-36238740817", 0.1805419921875], ["JD-35586107253", 0.1805419921875], ["JD-32584744058", 0.1805419921875], ["JD-531076", 0.179443359375], ["JD-531080", 0.178466796875], ["JD-12295977505", 0.177490234375], ["JD-100000008806", 0.1768798828125], ["JD-12295977507", 0.1768798828125], ["JD-39937044056", 0.176513671875], ["JD-39688051398", 0.176513671875], ["JD-39378786220", 0.176513671875]]

[0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0]
*/



