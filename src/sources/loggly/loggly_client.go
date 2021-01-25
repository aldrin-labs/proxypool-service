package loggly_client

import (
	"fmt"
	"log"
	"os"

	"github.com/segmentio/go-loggly"
)

type LogglyClient struct {
	Client *loggly.Client
}

var instance *LogglyClient

func GetInstance() *LogglyClient {
	if instance == nil {
		instance = &LogglyClient{}
		instance.init()
	}
	return instance
}

func (sd *LogglyClient) init() {
	// TODO: Add LOGGLY to env & secrets
	// host := os.Getenv("LOGGLY_TOKEN")

	environment := os.Getenv("ENVIRONMENT")

	sd.Client = loggly.New("86c8b2ca-742d-452e-99d6-030d862d6372", "proxypool-service", environment)
	log.Println("Loggly client init successful")
}

func (sd *LogglyClient) Info(a ...interface{}) {
	msg := fmt.Sprint(a...)
	log.Println(msg)
	if sd.Client != nil {
		err := sd.Client.Info(msg)
		if err != nil {
			log.Fatal(err)
		}
		sd.Client.Flush()
	}
}

func (sd *LogglyClient) Infof(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	log.Println(msg)
	if sd.Client != nil {
		err := sd.Client.Info(msg)
		if err != nil {
			log.Fatal(err)
		}
		sd.Client.Flush()
	}
}

func (sd *LogglyClient) Fatal(a ...interface{}) {
	msg := fmt.Sprint(a...)
	if sd.Client != nil {
		err := sd.Client.Info(msg)
		if err != nil {
			log.Fatal(err)
		}
		sd.Client.Flush()
	}
	log.Fatal(msg)
}
