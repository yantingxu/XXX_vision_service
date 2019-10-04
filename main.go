package main

import (
    "bytes"
    "errors"
    "flag"
	"fmt"
    "strings"
	"time"
    "net/http"
    "io/ioutil"

    "github.com/gin-gonic/gin"
)

const (
    serverReadTimeout   time.Duration   = 5 * time.Second
    serverWriteTimeout  time.Duration   = 5 * time.Second
    clientTimeout       time.Duration   = 5 * time.Second
    priceUrl            string          = "https://p.3.cn/prices/mgets"
    productUrl          string          = "http://192.168.1.11:8000/product"
    fashionUrl          string          = "http://192.168.1.11:8000/fashion"
    logoUrl             string          = "http://192.168.1.11:8000/logo"
    rerankUrl           string          = "http://192.168.1.11:8000/rerank"
    redisAddr           string          = "192.168.1.11:6379"
    TOPN                int             = 3
)

var (
    httpPort        string
    logFilename     string
)

func init() {
    flag.StringVar(&httpPort, "port", ":9070", "http server port")
    flag.StringVar(&logFilename, "log", "XXX-vision.app.log",  "application log filename")
}

// Input of the overall service
type inputParam struct {
    Code        string  `form:"code"`
    Content     string  `form:"content"`
    DbId        int     `form:"db_id"`
    ClassId     int     `form:"classid"`
    RandomValue string  `form:"randomValue"`
}

// Output of the overall service
type retItem struct {
    GID     string  `json:"gid"`
    Name    string  `json:"name"`
    Image   string  `json:"image"`
    Url     string  `json:"url"`
    Price   float64 `json:"price"`
    From    string  `json:"from"`
    Item    []string  `json:"item"`
}
type retItems []retItem

func assemble(rerank rerankResult, price priceResult, rds redisResult) retItems {
    var results retItems
    for i, pid := range rerank {
        ret := retItem {
            GID : pid,
            Name : rds[i].Name,
            Image : rds[i].Image,
            Url : rds[i].Url,
            Price : price[i].Price,
            From : strings.Split(pid, "-")[0],
            Item : []string{},
        }
        results = append(results, ret)
    }
    return results
}

// Handling Procedure
func handleShopping(c *gin.Context) {
    // save the body bytes for reuse
    bodyBytes, _ := ioutil.ReadAll(c.Request.Body)
    c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

    var param inputParam
    if err := c.ShouldBind(&param); err != nil {
        c.String(http.StatusOK, "failure")
        return
    }
    fmt.Println(param.Code)
    // fmt.Println(param.Content)

    var products productResult
    pChanOut := make(chan productResult)
    var fashion fashionResult
    fChanOut := make(chan fashionResult)

    var err error
    go handleProduct(bodyBytes, pChanOut)
    go handleFashion(bodyBytes, fChanOut)
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    defer close(pChanOut)
    defer close(fChanOut)

    t1 := time.Now()
loop1:
    for {
        select {
        case ps := <-pChanOut:
            if ps == nil {
                err = errors.New("Product Request Error")
                break loop1
            }
            products = ps
            if fashion != nil {
                break loop1
            }
        case fs := <-fChanOut:
            if fs == nil {
                err = errors.New("Fashion Request Error")
                break loop1
            }
            fashion = fs
            if products != nil {
                break loop1
            }
        case <-ticker.C:
            err = errors.New("Request Timeout")
            break loop1
        }
    }
    t2 := time.Since(t1).Seconds()
    fmt.Printf("Loop1 Time Elapsed: %fs\n", t2)

    if err != nil {
        fmt.Println(err)
        c.String(http.StatusOK, err.Error())
        return
    }
    // fmt.Printf("Product: %v\n", *products)
    // fmt.Printf("Fashion: %v\n", *fashion)

    var rerank rerankResult
    rChanOut := make(chan rerankResult)
    var price priceResult
    prChanOut := make(chan priceResult)
    var rds redisResult
    rdsChanOut := make(chan redisResult)

    go handleRerank(products, fashion, param.ClassId, param.RandomValue, rChanOut)
    go handlePrice(products, prChanOut)
    go handleRedis(products, rdsChanOut)
    ticker2 := time.NewTicker(10 * time.Second)
    defer ticker2.Stop()
    defer close(rChanOut)
    defer close(prChanOut)
    defer close(rdsChanOut)

    t1 = time.Now()
loop2:
    for {
        select {
        case rs := <-rChanOut:
            if rs == nil {
                err = errors.New("Rerank Request Error")
                break loop2
            }
            rerank = rs
            if price != nil && rds != nil{
                break loop2
            }
        case ps := <-prChanOut:
            if ps == nil {
                err = errors.New("Price Request Error")
                break loop2
            }
            price = ps
            if rerank != nil && rds != nil {
                break loop2
            }
        case rs := <-rdsChanOut:
            if rs == nil {
                err = errors.New("Redis Request Error")
                break loop2
            }
            rds = rs
            if rerank != nil && price != nil {
                break loop2
            }
        case <-ticker.C:
            err = errors.New("Request Timeout")
            break loop2
        }
    }
    t2 = time.Since(t1).Seconds()
    fmt.Printf("Loop2 Time Elapsed: %fs\n", t2)

    if err != nil {
        fmt.Println(err)
        c.String(http.StatusOK, err.Error())
        return
    }
    // fmt.Printf("Rerank: %v\n", rerank)
    // fmt.Printf("Price: %v\n", *price)
    // fmt.Printf("Redis: %v\n", *rds)

    retItems := assemble(rerank, price, rds)
    c.JSON(http.StatusOK, gin.H {
        "code" : 101,
        "msg" : param.RandomValue,
        "data" : retItems[:TOPN],
    })
}

func handleShoppingFake(c *gin.Context) {
    // save the body bytes for reuse
    bodyBytes, _ := ioutil.ReadAll(c.Request.Body)
    c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

    fakeRetItems := []retItem {
        retItem {
            GID : "JD-30034871415",
            Name : "【沃尔玛】益达 木糖醇无糖口香糖冰凉薄荷味 56g",
            Image : "http://img10.360buyimg.com/n4/jfs/t21799/13/2092184476/471040/1437debb/5b488455N55a4273b.png",
            Url : "https://item.jd.com/30034871415.html",
            Price : 8.9,
            From : "JD",
            Item : []string{},
        },
        retItem {
            GID : "JD-39688051397",
            Name : "箭牌 益达木糖醇无糖口香糖 56g瓶装 约40粒装 休闲零食 冰凉薄荷味",
            Image : "http://img12.360buyimg.com/n4/jfs/t3673/290/1681006231/306576/171749bc/582ebc2fN8b8ee327.jpg",
            Url : "https://item.jd.com/39688051397.html",
            Price : 11.5,
            From : "JD",
            Item : []string{},
        },
        retItem {
            GID : "JD-39378786216",
            Name : "箭牌 益达木糖醇无糖口香糖 56g瓶装 约40粒装 休闲零食 冰凉薄荷味",
            Image : "http://img11.360buyimg.com/n4/jfs/t3673/290/1681006231/306576/171749bc/582ebc2fN8b8ee327.jpg",
            Url : "https://item.jd.com/39378786216.html",
            Price : 11.5,
            From : "JD",
            Item : []string{},
        },
    }
    c.JSON(http.StatusOK, gin.H {
        "code" : 101,
        "msg" : "123",
        "data" : fakeRetItems[:TOPN],
    })
    return
}

func main() {
    // Parse Command-line Args
    flag.Parse()

    // Run in Release Mode
    gin.SetMode(gin.ReleaseMode)

    // Create Redis Client Pool
	pool = newPool(redisAddr)

    // Create Router with Customized Logging Format
	r := gin.New()

    /*
    gin.DisableConsoleColor()
    f, _ := os.Create(logFilename)
    gin.DefaultWriter = io.MultiWriter(f, os.Stdout)
    */
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
				param.ClientIP,
				param.TimeStamp.Format(time.RFC1123),
				param.Method,
				param.Path,
				param.Request.Proto,
				param.StatusCode,
				param.Latency,
				param.Request.UserAgent(),
				param.ErrorMessage,
		)
	}))
    r.Use(gin.Recovery())

    // Ping (for testing)
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H {
			"message": "pong",
		})
	})

    // Router
    r.POST("/shopping", handleShopping)

	// Start Serving with Customized Timeout Config
	s := &http.Server {
		Addr:           httpPort,
		Handler:        r,
		ReadTimeout:    serverReadTimeout,
		WriteTimeout:   serverWriteTimeout,
	}
	s.ListenAndServe()
}




