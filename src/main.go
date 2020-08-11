package main

import (
	"github.com/joho/godotenv"
	"gitlab.com/crypto_project/core/proxypool_service/src/api"
)

func main() {
	godotenv.Load("../.env")
	api.RunServer()
}
