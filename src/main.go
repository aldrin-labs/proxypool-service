package main

import (
	"github.com/joho/godotenv"
	"gitlab.com/crypto_project/core/proxypool_service/src/api"
)

var port = ":5901"

func main() {
	godotenv.Load()
	api.RunServer(port)
}
