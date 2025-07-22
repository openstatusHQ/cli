package monitors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/openstatusHQ/cli/internal/config"
)

func UpdateMonitor(httpClient *http.Client, apiKey string, id int, monitor config.Monitor) (Monitor, error) {

	url := fmt.Sprintf("https://api.openstatus.dev/v1/monitor/%s/%d", monitor.Kind, id)

	payloadBuf := new(bytes.Buffer)
	json.NewEncoder(payloadBuf).Encode(monitor)
	req, _ := http.NewRequest(http.MethodPut, url, payloadBuf)

	req.Header.Add("x-openstatus-key", apiKey)
	req.Header.Add("Content-Type", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		return Monitor{}, err
	}

	if res.StatusCode != http.StatusOK {
		return Monitor{}, fmt.Errorf("Failed to Update monitor")
	}

	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	var monitors Monitor
	err = json.Unmarshal(body, &monitors)
	if err != nil {
		return Monitor{}, err
	}

	return monitors, nil
}
