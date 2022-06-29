package flare

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

func SendTelemetry(domain, metricName string) {

	url := "https://metaphor-go-production.kubefirst.io/telemetry"
	method := "POST"

	payload := strings.NewReader(fmt.Sprintf(`{"domain": "%s","name": "%s"}`, domain, metricName))

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
	}

	req.Header.Add("Content-Type", "application/json")
	// TODO need to add authentication or a header of some sort?
	// req.Header.Add("auth?", os.Getenv("K1_KEY"))

	res, err := client.Do(req)
	if err != nil {
		fmt.Println("error")
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)

	fmt.Println(string(body))
}
