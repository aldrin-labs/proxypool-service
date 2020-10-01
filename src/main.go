package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"gitlab.com/crypto_project/core/proxypool_service/src/api"
	"gitlab.com/crypto_project/core/proxypool_service/src/healthcheck"
)

var port = ":5901"

func main() {
	godotenv.Load()

	// wait group keeps main process running
	var wg sync.WaitGroup

	go shutdownHandler(&wg)

	go api.RunServer(port)
	wg.Add(1)

	go healthcheck.RunProxiesHealthcheck()

	wg.Wait()
}

// sleep for N seconds on instance shutdown signal
// to make sure we finish serving all clients before actual shutdown
func shutdownHandler(wg *sync.WaitGroup) {
	sigint := make(chan os.Signal, 1)

	// interrupt signal sent from terminal
	signal.Notify(sigint, os.Interrupt)
	// sigterm signal sent from kubernetes
	signal.Notify(sigint, syscall.SIGTERM)

	// waiting for shutdown signal
	<-sigint

	log.Printf("Starting instance shutdown...")

	time.Sleep(10 * time.Second)

	log.Printf("Wait group done...")
	wg.Done()
}
