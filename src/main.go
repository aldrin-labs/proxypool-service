package main

import (
	"fmt"
	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
	"log"
)


func GetProxy(ctx *fasthttp.RequestCtx) {
	fmt.Fprint(ctx, "GetProxy!\n")

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
