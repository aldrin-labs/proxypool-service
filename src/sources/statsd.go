package sources

import (
	"fmt"
	loggly_client "gitlab.com/crypto_project/core/proxypool_service/src/sources/loggly"

	"github.com/cactus/go-statsd-client/statsd"
)

type StatsdClient struct {
	Client *statsd.Statter
}

func (sd *StatsdClient) Init() {
	// host := os.Getenv("STATSD_HOST")
	host := ""
	if host == "" {
		loggly_client.GetInstance().Infof("Warning. Hostname for statsd is empty. Using default one.")
		host = "statsd.infra"
	}
	port := "8125"

	loggly_client.GetInstance().Infof("Statsd connecting to %s:%s...", host, port)

	config := &statsd.ClientConfig{
		Address: fmt.Sprintf("%s:%s", host, port),
		Prefix:  fmt.Sprintf("proxy_pool"),
	}

	client, err := statsd.NewClientWithConfig(config)
	if err != nil {
		loggly_client.GetInstance().Info("Error on Statsd init:" + err.Error())
		return
	}

	sd.Client = &client

	loggly_client.GetInstance().Info("Statsd init successful")
}

func (sd *StatsdClient) Inc(statName string) {
	if sd.Client != nil {
		err := (*sd.Client).Inc(statName, 1, 1.0)
		if err != nil {
			loggly_client.GetInstance().Info("Error on Statsd Inc:" + err.Error())
		}
	}
}

func (sd *StatsdClient) IncBy(statName string, value int64) {
	if sd.Client != nil {
		err := (*sd.Client).Inc(statName, value, 1.0)
		if err != nil {
			loggly_client.GetInstance().Info("Error on Statsd Inc:" + err.Error())
		}
	}
}

func (sd *StatsdClient) Timing(statName string, value int64) {
	if sd.Client != nil {
		err := (*sd.Client).Timing(statName, value, 1.0)
		if err != nil {
			loggly_client.GetInstance().Info("Error on Statsd Timing:" + err.Error())
		}
	}
}

func (sd *StatsdClient) Gauge(statName string, value int64) {
	if sd.Client != nil {
		err := (*sd.Client).Gauge(statName, value, 1.0)
		if err != nil {
			loggly_client.GetInstance().Info("Error on Statsd Gauge:" + err.Error())
		}
	}
}
