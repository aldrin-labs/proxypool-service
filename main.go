package main

import (
	"fmt"
	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
	"log"
	"gitlab.com/crypto_project/core/proxypool_service/src/proxypool"
)


func GetProxy(ctx *fasthttp.RequestCtx) {
	exchange := string(ctx.QueryArgs().Peek("exchange"))
	_, _ = fmt.Fprint(ctx, proxypool.GetProxyPool().GetProxyByExchange(exchange))
}

// Index is the index handler
func Index(ctx *fasthttp.RequestCtx) {
	fmt.Fprint(ctx, "Welcome!\n")
}

func main() {
	router := fasthttprouter.New()

	router.GET("/", Index)
	router.GET("/getProxy", GetProxy)

	log.Fatal(fasthttp.ListenAndServe(":5901", router.Handler))
}
