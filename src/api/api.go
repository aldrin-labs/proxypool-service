package api

import (
	"encoding/json"
	"fmt"
	"github.com/buaazp/fasthttprouter"
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

	log.Printf("Got GetProxyByPriority request with %d priority from %s", priority, ctx.RemoteIP())

	jsonStr, _ := json.Marshal(pool.GetProxyPoolInstance().GetProxyByPriority(priority))
	_, _ = fmt.Fprint(ctx, string(jsonStr))
}

func TestProxy(ctx *fasthttp.RequestCtx) {
	for i := 0; i < 300; i++ {
		go func() {
			url := pool.GetProxyPoolInstance().GetTopPriorityProxy()
			time.Sleep(10 * time.Millisecond)
			pool.GetProxyPoolInstance().ExtemptProxy(url)
		}()

	}
}

// Index is the index handler
func Index(ctx *fasthttp.RequestCtx) {
	proxyPool := pool.GetProxyPoolInstance()
	data := fmt.Sprintf("%v", proxyPool.GetStats())
	fmt.Fprint(ctx, data)
}

func Extempt(ctx *fasthttp.RequestCtx) {
	println("call ex")
	res := &struct {
		Proxy string
	}{}
	err := json.Unmarshal(ctx.PostBody(), res)

	if err != nil {
		log.Print("err while Extempt", err.Error())
	}

	pool.GetProxyPoolInstance().ExtemptProxy(res.Proxy)
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
	router.POST("/extempt", Extempt)

	log.Print("Listening on port :5901")
	log.Fatal(fasthttp.ListenAndServe(":5901", router.Handler))
}
