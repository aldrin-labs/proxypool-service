package main

import (
	"encoding/json"
	"fmt"
	"github.com/buaazp/fasthttprouter"
	"github.com/joho/godotenv"
	"github.com/valyala/fasthttp"
	"gitlab.com/crypto_project/core/proxypool_service/src/pool"
	"log"
	"strconv"
	"time"
)

func GetProxy(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("application/json; charset=utf8")

	priority, err := strconv.Atoi(string(ctx.QueryArgs().Peek("priority")))
	if err != nil {
		fmt.Println("error:", err)
		priority = 1
	}
	jsonStr, _ := json.Marshal(pool.GetProxyPoolInstance().GetProxyByPriority(priority))
	_, _ = fmt.Fprint(ctx, string(jsonStr))
}

func TestProxy(ctx *fasthttp.RequestCtx) {
	prev := time.Now()
	proxyForTest := pool.GetProxyPoolInstance().Proxies[0][0]
	for i := 0; i < 300; i++ {
		now := pool.GetProxyPoolInstance().ExchangeProxyMap[0][proxyForTest].RateLimiter.Take()
		fmt.Println(i, now.Sub(prev))
		prev = now
	}
}

// Index is the index handler
func Index(ctx *fasthttp.RequestCtx) {
	fmt.Fprint(ctx, "Welcome!\n")
}

func Healthz(ctx *fasthttp.RequestCtx) {
	fmt.Fprint(ctx, "alive!\n")
}

func main() {
	godotenv.Load()

	router := fasthttprouter.New()
	pool.GetProxyPoolInstance()
	router.GET("/", Index)
	router.GET("/getProxy", GetProxy)
	router.GET("/testProxy", TestProxy)
	router.GET("/healthz", Healthz)

	log.Print("Listening on port :5901")
	log.Fatal(fasthttp.ListenAndServe(":5901", router.Handler))
}
