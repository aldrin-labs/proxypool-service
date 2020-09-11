package api

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
	"gitlab.com/crypto_project/core/proxypool_service/src/pool"
)

func GetProxy(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("application/json; charset=utf8")

	priority, err := strconv.Atoi(string(ctx.QueryArgs().Peek("priority")))
	weight, err := strconv.Atoi(string(ctx.QueryArgs().Peek("weight")))
	if err != nil {
		fmt.Println("error:", err)
		priority = 1
	}

	log.Printf("Got GetProxyByPriority request with %d priority and %d weight from %s", priority, weight, ctx.RemoteIP())

	jsonStr, _ := json.Marshal(pool.GetProxyPoolInstance().GetProxyByPriority(priority, weight))
	_, _ = fmt.Fprint(ctx, string(jsonStr))
}

func TestProxy(ctx *fasthttp.RequestCtx) {
	weight, _ := strconv.Atoi(string(ctx.QueryArgs().Peek("weight")))
	for i := 0; i < 300; i++ {
		go func(i int) {
			proxyRes := pool.GetProxyPoolInstance().GetTopPriorityProxy(weight)
			time.Sleep(10 * time.Millisecond)
			pool.GetProxyPoolInstance().ExemptProxy(proxyRes.Proxy, i)
		}(i)
	}
}

// Index is the index handler
func Index(ctx *fasthttp.RequestCtx) {
	proxyPool := pool.GetProxyPoolInstance()
	data := fmt.Sprintf("%v", proxyPool.GetStats())
	fmt.Fprint(ctx, data)
}

func Exempt(ctx *fasthttp.RequestCtx) {
	// println("call ex")
	res := &struct {
		Proxy   string `json:"proxy"`
		Counter int    `json:"counter"`
	}{}
	err := json.Unmarshal(ctx.PostBody(), res)

	// fmt.Printf("%s", ctx.PostBody())

	if err != nil {
		log.Print("err while Exempt", err.Error())
	}

	pool.GetProxyPoolInstance().ExemptProxy(res.Proxy, res.Counter)
}

func Healthz(ctx *fasthttp.RequestCtx) {
	fmt.Fprint(ctx, "alive!\n")
}

func RunServer() {
	router := fasthttprouter.New()
	pool.GetProxyPoolInstance()
	router.GET("/", Index)
	router.GET("/getProxy", GetProxy)
	router.GET("/testProxy", TestProxy)
	router.GET("/healthz", Healthz)
	router.POST("/exempt", Exempt)

	log.Print("Listening on port :5901")
	log.Fatal(fasthttp.ListenAndServe(":5901", router.Handler))
}
