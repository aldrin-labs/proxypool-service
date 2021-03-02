# core/proxypool_service
Returns proxy URL and credentials on request. Short name - `PP`.

## Description
Proxy pool returns proxy credentials, it does not proxy requests. Service also does rate limiting.   

## How it works
(request from client) -> PP API -> PP pool logic -> Redis [rate limiter] -> PP pool logic -> (response to client)

0) On startup service reads proxy lists from ENV and creates `ProxyPool` from it in memory
1) Any service makes request to PP API with priority and weight of planned call to binance.
2) API handles request to `ProxyPool` singleton
3) `ProxyPool` selects one proxy from list using some algo (now `Round Robin`)
4) `ProxyPool` calls redis using special rate-limiting library with data about selected proxy and planned weight
5) `Redis` says if proxy can be served right away or should be throttled (with time needed)
6) Service serves proxy to the client

Throttling means some delay before proxy can be returned to client.

## Features and implementation details
1) Rate limiting   
During service life we tried to use different rate limiting solutions: golang's native time.Limiter, 
uber's uber-go/ratelimit and go-redis/redis_rate which uses redis.   
Redis solution makes scaling possible.

2) Proxy healthcheck   
Service makes request using proxies periodically.

3) Priority   
Service can serve proxies from different lists grouped by proxy priority. 0 is highest priority proxies. Service can switch request to
lower priority proxy if all high priority proxies are blocked.

## What can be improved
1) Separate calls to binance futures and binance spot api's. These APIs have separate request weight counters on binance side, but we
don't distinguish them.
2) Find a way to check if proxy is over limit without adding weight to internal rate-limiter counter

## Local deployment and building

Get sources for the service and dependencies

`go get -u -d -v gitlab.com/crypto_project/core/signal_service.git`

Latest versions is ok for now, might want to use dep and versioning later

To build the executables, you might want to use either

`go build`

in the project's root, which will build an executable right to the place where you invoked it, or

`go install`

which will build an executable to your $gopath/bin

## Tests

> go test ./tests
> go test -v ./tests/http_test.go