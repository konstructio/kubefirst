package telemetry

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

// SendTelemetry post telemetry data
func SendTelemetry(useTelemetry bool, domain string, metricName string) {
    defer func() {
        if r := recover(); r != nil {
            log.Println("Error sending telemetry. Error:\n", r)
        }
    }()

	if !useTelemetry {
		log.Println("Telemetry disable by user choice, nothing was sent")
		return
	}

	log.Println("SendTelemetry (working...)")

	url := "https://metaphor-go-production.kubefirst.io/telemetry"
	method := "POST"

	payload := strings.NewReader(fmt.Sprintf(`{"domain": "%s","name": "%s"}`, domain, metricName))

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		log.Println(err)
	}

	req.Header.Add("Content-Type", "application/json")
	// TODO need to add authentication or a header of some sort?
	// req.Header.Add("auth?", os.Getenv("K1_KEY"))

	res, err := client.Do(req)
	if err != nil {
		log.Println("error")
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println("error")
	}

	log.Println(string(body))

	log.Println("SendTelemetry (done)")
}
