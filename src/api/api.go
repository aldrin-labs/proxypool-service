package api

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

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
}

func TestProxies(ctx *fasthttp.RequestCtx) {
	results := make(map[string]pool.HealthCheckResponse)

	pp := pool.GetProxyPoolInstance()
	proxies := pp.Proxies
	for priority := range proxies {
		for _, proxyURL := range proxies[priority] {
			translatedProxyURL, err := pool.TranslateProxyNameToProxyURL(proxyURL)
			if err != nil {
				log.Printf("Error while translating proxy url %s", proxyURL)
				continue
			}

			checkResult := pool.CheckProxy(translatedProxyURL)
			results[proxyURL] = checkResult

			// log.Printf("%v : %v", proxyURL, checkResult)
		}
	}

	jsonStr, err := json.Marshal(results)
	if err != nil {
		return
	}

	fmt.Fprint(ctx, string(jsonStr))
}

// Index is the index handler
func Index(ctx *fasthttp.RequestCtx) {
	proxyPool := pool.GetProxyPoolInstance()
	data := fmt.Sprintf("%v", proxyPool.GetStats())
	fmt.Fprint(ctx, data)
}

func Exempt(ctx *fasthttp.RequestCtx) {
	// println("called Exempt")
}

func Healthz(ctx *fasthttp.RequestCtx) {
	fmt.Fprint(ctx, "alive!\n")
}

func RunServer(port string) {
	router := fasthttprouter.New()
	pool.GetProxyPoolInstance()
	router.GET("/", Index)
	router.GET("/getProxy", GetProxy)
	router.GET("/testProxy", TestProxy)
	router.GET("/testProxies", TestProxies)
	router.GET("/healthz", Healthz)
	router.POST("/exempt", Exempt)

	log.Printf("Listening on port %s", port)
	log.Fatal(fasthttp.ListenAndServe(port, router.Handler))
}
