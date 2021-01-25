package sources

import (
	loggly_client "gitlab.com/crypto_project/core/proxypool_service/src/sources/loggly"
	"os"
	"time"

	"gitlab.com/crypto_project/core/proxypool_service/src/helpers"
)

// Prometheus AlertManager APIv2 methods:
// https://petstore.swagger.io/?url=https://raw.githubusercontent.com/prometheus/alertmanager/master/api/v2/openapi.yaml

var paManager = promAlertManager{}

var serviceName = "proxypool-service"
var generatorURL = "http://proxypool-service.proxypool-service.svc.cluster.local"

type promAlertManager struct {
	Url     string
	ApiPath string
}

type alertRequest struct {
	Alerts []alert `json:"alerts"`
}

// StartsAt and EndsAt date format - 2020-09-16T15:15:56.070Z
type alert struct {
	StartsAt     string            `json:"startsAt"`
	EndsAt       string            `json:"endsAt",`
	Annotations  map[string]string `json:"annotations"`
	Labels       map[string]string `json:"labels"`
	GeneratorURL string            `json:"generatorURL"`
}

func GetPrometheusNotifierInstance() *promAlertManager {
	if paManager.Url == "" {
		paManager.Url = os.Getenv("ALERT_MANAGER")
		if paManager.Url == "" {
			paManager.Url = "http://prom-operator-prometheus-o-alertmanager.infra.svc.cluster.local:9093"
			// paManager.Url = "http://localhost:9093"
		}

		paManager.ApiPath = "/api/v2"
	}

	return &paManager
}

func (am *promAlertManager) Notify(message string, source string) {

	startDate := time.Now().Format(time.RFC3339)
	endDate := time.Now().Add(5 * time.Minute).Format(time.RFC3339)

	requestParams := []alert{
		{
			StartsAt: startDate,
			EndsAt:   endDate,
			Annotations: map[string]string{
				"message": message,
			},
			Labels: map[string]string{
				"service": serviceName,
				"source":  source,
			},
			GeneratorURL: generatorURL,
		},
	}

	result, err := helpers.MakePostRequest(am.Url+am.ApiPath, "alerts", requestParams)
	_, _ = result, err

	// Error %!(EXTRA string=unexpected end of JSON input) is fine, it just returns empty response even if alert was created
	// if err != nil {
	// 	loggly_client.GetInstance().Infof("[Notify] Error: ", err.Error())
	// }
	// loggly_client.GetInstance().Infof("%v", result)

	loggly_client.GetInstance().Infof("[Notify] Message sent from %s", source)
}

func (am *promAlertManager) GetStatus() {
	result, err := helpers.MakeGetRequest(am.Url+am.ApiPath, "status", nil)
	if err != nil {
		loggly_client.GetInstance().Infof("Error: %s", err.Error())
	}
	loggly_client.GetInstance().Infof("%v", result)
}
