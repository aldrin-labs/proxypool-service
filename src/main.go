package main

import (
	"fmt"
	"github.com/buaazp/fasthttprouter"
	"github.com/joho/godotenv"
	"github.com/valyala/fasthttp"
	"gitlab.com/crypto_project/core/proxypool_service/src/pool"
	"log"
)


func GetProxy(ctx *fasthttp.RequestCtx) {
	exchange := string(ctx.QueryArgs().Peek("exchange"))

	_, _ = fmt.Fprint(ctx, pool.GetProxyPoolInstance().GetProxyByExchange(exchange))
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
