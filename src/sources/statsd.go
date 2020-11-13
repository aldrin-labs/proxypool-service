package sources

import (
	"fmt"
	"log"
	"os"

	"github.com/cactus/go-statsd-client/statsd"
)

type StatsdClient struct {
	Client *statsd.Statter
}

func (sd *StatsdClient) Init() {
	host := os.Getenv("STATSD_HOST")
	if host == "" {
		log.Printf("Warning. Hostname for statsd is empty. Using default one.")
		host = "graphite.infra"
	}
	port := "8125"

	log.Printf("Statsd connecting to %s:%s...", host, port)

	config := &statsd.ClientConfig{
		Address: fmt.Sprintf("%s:%s", host, port),
		Prefix:  fmt.Sprintf("proxy_pool"),
	}

	client, err := statsd.NewClientWithConfig(config)
	if err != nil {
		log.Println("Error on Statsd init:" + err.Error())
		return
	}

	sd.Client = &client

	log.Println("Statsd init successful")
}

func (sd *StatsdClient) Inc(statName string) {
	if sd.Client != nil {
		err := (*sd.Client).Inc(statName, 1, 1.0)
		if err != nil {
			log.Println("Error on Statsd Inc:" + err.Error())
		}
	}
}

func (sd *StatsdClient) Timing(statName string, value int64) {
	if sd.Client != nil {
		err := (*sd.Client).Timing(statName, value, 1.0)
		if err != nil {
			log.Println("Error on Statsd Timing:" + err.Error())
		}
	}
}

func (sd *StatsdClient) Gauge(statName string, value int64) {
	if sd.Client != nil {
		err := (*sd.Client).Gauge(statName, value, 1.0)
		if err != nil {
			log.Println("Error on Statsd Gauge:" + err.Error())
		}
	}
}
