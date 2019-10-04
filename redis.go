package main

import (
    "fmt"
    "time"
    "strconv"

    "github.com/gomodule/redigo/redis"
)

var pool *redis.Pool
func newPool(addr string) *redis.Pool {
	return &redis.Pool {
		IdleTimeout: 240 * time.Second,
		Dial: func () (redis.Conn, error) {
            return redis.Dial("tcp", addr)
        },
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
}

// Redis
type productExt struct {
    Name        string
    Url         string
    Price       float64
    Image       string
}
type redisResult []productExt

func handleRedisFake(products productResult, chanOut chan redisResult) redisResult {
    p1 := productExt {
        Name : "【沃尔玛】益达 木糖醇无糖口香糖冰凉薄荷味 56g",
        Image : "http://img10.360buyimg.com/n4/jfs/t21799/13/2092184476/471040/1437debb/5b488455N55a4273b.png",
        Url : "https://item.jd.com/30034871415.html",
        Price : 8.9,
    }
    p2 := productExt {
        Name : "箭牌 益达木糖醇无糖口香糖 56g瓶装 约40粒装 休闲零食 冰凉薄荷味",
        Image : "http://img12.360buyimg.com/n4/jfs/t3673/290/1681006231/306576/171749bc/582ebc2fN8b8ee327.jpg",
        Url : "https://item.jd.com/39688051397.html",
        Price : 11.5,
    }
    p3 := productExt {
        Name : "箭牌 益达木糖醇无糖口香糖 56g瓶装 约40粒装 休闲零食 冰凉薄荷味",
        Image : "http://img11.360buyimg.com/n4/jfs/t3673/290/1681006231/306576/171749bc/582ebc2fN8b8ee327.jpg",
        Url : "https://item.jd.com/39378786216.html",
        Price : 11.5,
    }
    // productExts := []productExt{p1, p2, p3}
    productExts := redisResult{p1, p2, p3}
    chanOut <- productExts
    return productExts
}

func handleRedis(products productResult, chanOut chan redisResult) redisResult {
    conn := pool.Get()
    defer conn.Close()

    // pids := []string{"JD-30034871415", "JD-39688051397"}
    t1 := time.Now()
    for _, product := range products {
        conn.Send("HMGET", product.PID, "name", "url", "price", "image")
    }
    conn.Flush()
    t2 := time.Since(t1).Seconds()
    fmt.Printf("HandleRedis Time Elapsed: %fs\n", t2)

    var productExts redisResult
    var name, url, price, image string
    for i := range products {
        vals, err := redis.Values(conn.Receive())
        if err != nil {
            fmt.Println(err)
            continue
        }
        vals, err = redis.Scan(vals, &name, &url, &price, &image)
        if err != nil {
            fmt.Println(err)
            continue
        }
        priceFloat, _ := strconv.ParseFloat(price, 64)
        product := productExt{name, url, priceFloat, image}
        productExts = append(productExts, product)
        if i == 0 {
            fmt.Printf("%d\t%s\t%s\t%s\t%s\n", i, name, url, price, image)
        }
    }
    chanOut <- productExts
    return productExts
}

