package main

import (
	"github.com/joho/godotenv"
	"gitlab.com/crypto_project/core/signal_service/src/server"
	"gitlab.com/crypto_project/core/signal_service/src/signals"
	"sync"
)

func main() {
	godotenv.Load()
	var wg sync.WaitGroup
	//TODO: init top-level context
	//notif := filtering.NewNotifier()
	//log.Println(notif)
	//sub := mongodb.NewSubscription(notif, "ccai-dev", "notifications2")
	//go sub.RunDataPull()
	//log.Println(err)
	//redisSub := redis.NewSubscription(notif)
	//go redisSub.RunDataPull()
	wg.Add(1)
	go server.RunServer(&wg)
	wg.Add(1)
	go signals.GetInstance().Init(&wg)
	wg.Wait()
}
