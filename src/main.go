package main

import (
	"encoding/json"
	"fmt"
	"github.com/buaazp/fasthttprouter"
	"github.com/joho/godotenv"
	"github.com/valyala/fasthttp"
	"gitlab.com/crypto_project/core/proxypool_service/src/pool"
	"log"
)


func GetProxy(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("application/json; charset=utf8")

	exchange := string(ctx.QueryArgs().Peek("exchange"))
	jsonStr, _ := json.Marshal(pool.GetProxyPoolInstance().GetProxyByExchange(exchange))
	_, _ = fmt.Fprint(ctx, string(jsonStr))
}

// Index is the index handler
func Index(ctx *fasthttp.RequestCtx) {
	fmt.Fprint(ctx, "Welcome!\n")
}

func main() {
	godotenv.Load()

	router := fasthttprouter.New()
	pool.GetProxyPoolInstance()
	router.GET("/", Index)
	router.GET("/getProxy", GetProxy)

	log.Fatal(fasthttp.ListenAndServe(":5901", router.Handler))
}
