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

	url := fmt.Sprintf("%s/monitor/%s/%d", APIBaseURL, monitor.Kind, id)

	payloadBuf := new(bytes.Buffer)
	json.NewEncoder(payloadBuf).Encode(monitor)
	req, err := http.NewRequest(http.MethodPut, url, payloadBuf)
	if err != nil {
		return Monitor{}, fmt.Errorf("failed to create request: %w", err)
	}

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
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return Monitor{}, fmt.Errorf("failed to read response body: %w", err)
	}

	var monitors Monitor
	err = json.Unmarshal(body, &monitors)
	if err != nil {
		return Monitor{}, err
	}

	return monitors, nil
}
