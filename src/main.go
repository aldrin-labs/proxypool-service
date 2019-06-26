package main

import (
	"fmt"
	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
	"log"
)

type proxyMap struct {
	proxymap map[string] int
}

type exchangeMap struct {
	exchangemap map[string] proxyMap

}

type proxyService struct {
	interpreter int
	proxyS []string
	ExchangeMap exchangeMap
	exchangeLimits
}


type exchangeLimits struct {
	exchangeNames map[string] uint64
}



func GetProxy(ctx *fasthttp.RequestCtx) {
	fmt.Fprint(ctx, "Welcome!\n")
}

func Hello(ctx *fasthttp.RequestCtx) {
	fmt.Fprintf(ctx, "hello, %s!\n", ctx.UserValue("name"))
}

func main() {
	router := fasthttprouter.New()
	router.GET("/getProxy", Index)

	log.Fatal(fasthttp.ListenAndServe(":8080", router.Handler))
}
