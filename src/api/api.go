package api

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	loggly_client "gitlab.com/crypto_project/core/proxypool_service/src/sources/loggly"

	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
	"gitlab.com/crypto_project/core/proxypool_service/src/healthcheck"
	"gitlab.com/crypto_project/core/proxypool_service/src/pool"
)

func GetProxy(ctx *fasthttp.RequestCtx) {
	start := time.Now()
	ctx.SetContentType("application/json; charset=utf8")

	dest := string(ctx.QueryArgs().Peek("destination"))
	if dest == "" {
		dest = "undefined"
	}
	priority, err := strconv.Atoi(string(ctx.QueryArgs().Peek("priority")))
	weight, err := strconv.Atoi(string(ctx.QueryArgs().Peek("weight")))
	if err != nil {
		loggly_client.GetInstance().Info("error:", err)
		priority = 1
	}

	pp := pool.GetProxyPoolInstance()
	proxy := pp.GetProxyByPriority(dest, priority, weight)

	if weight > 50 {
		loggly_client.GetInstance().Infof("Got GetProxyByPriority request with %d priority and %d weight from %s for %s", priority, weight, ctx.RemoteIP(), dest)
	}

	// prepare response
	jsonBytes, _ := json.Marshal(proxy)
	jsonString := string(jsonBytes)

	// measure response time
	duration := time.Since(start)
	pp.GetMetricsClient().Timing("api.getProxy.duration", int64(duration.Milliseconds()))

	_, _ = fmt.Fprint(ctx, jsonString)
}

func TestProxy(ctx *fasthttp.RequestCtx) {
}

func TestProxies(ctx *fasthttp.RequestCtx) {
	results := make(map[string]healthcheck.HealthCheckResponse)

	ch := make(chan healthcheck.HealthCheckResponse)

	pp := pool.GetProxyPoolInstance()
	proxies := pp.Proxies

	proxyHttpClients := healthcheck.CreateProxyHttpClients(proxies)

	numberRequests := 0
	for priority := range proxies {
		for _, proxyURL := range proxies[priority] {
			proxyHttpClient := proxyHttpClients[priority][proxyURL]

			go healthcheck.CheckProxy(proxyURL, proxyHttpClient, priority, ch)
			numberRequests++
		}
	}

	// getting results
	for i := 1; i <= numberRequests; i++ {
		checkResult := <-ch
		proxyURL := checkResult.ProxyURL
		results[proxyURL] = checkResult
		// loggly_client.GetInstance().Infof("%v : %v", proxyURL, checkResult)
	}

	jsonStr, err := json.Marshal(results)
	if err != nil {
		return
	}

	fmt.Fprint(ctx, string(jsonStr))
}

func Index(ctx *fasthttp.RequestCtx) {
	proxyPool := pool.GetProxyPoolInstance()
	data := fmt.Sprintf("%v", proxyPool.GetStats())
	fmt.Fprint(ctx, data)
}

func Exempt(ctx *fasthttp.RequestCtx) {
	// loggly_client.GetInstance().Info("called Exempt")
}

func markProxyUnhealthy(ctx *fasthttp.RequestCtx) {
	params := &struct {
		ProxyURL string
		Priority int
	}{}
	err := json.Unmarshal(ctx.PostBody(), params)

	if err != nil {
		loggly_client.GetInstance().Info("Error while parsing POST params: ", err.Error())
		fmt.Fprint(ctx, "{\"status\": \"ERR\"}")
		return
	}

	pool.GetProxyPoolInstance().MarkProxyAsUnhealthy(params.Priority, params.ProxyURL)
	fmt.Fprint(ctx, "{\"status\": \"OK\"}")
}

func GetUnhealthyProxies(ctx *fasthttp.RequestCtx) {
	proxyPool := pool.GetProxyPoolInstance()

	var data string
	for _, proxies := range proxyPool.ExchangeProxyMap {
		for _, proxy := range proxies {
			if !proxy.Healthy {
				data += fmt.Sprintf("Proxy %s is unhealthy since %d \n", proxy.URL, proxy.HealthStatusLastChange)
			}
		}
	}
	fmt.Fprint(ctx, data)
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
	router.GET("/getUnhealthy", GetUnhealthyProxies)
	router.GET("/healthz", Healthz)
	router.POST("/exempt", Exempt)
	router.POST("/markProxyUnhealthy", markProxyUnhealthy)

	loggly_client.GetInstance().Infof("Listening on port %s", port)
	loggly_client.GetInstance().Fatal(fasthttp.ListenAndServe(port, router.Handler))
}
