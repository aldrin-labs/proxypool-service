# Testing proxypool_service

## Prerequisites

You will need

* redis running on your machine as it uses redis for rate limiting
* .env file placed in ./tests folder with PROXYLIST (base64), REDIS_PASSWORD, REDIS_HOST, REDIS_PORT variables

## Running tests

> go test -v ./tests   
> go test -v ./tests/http_test.go