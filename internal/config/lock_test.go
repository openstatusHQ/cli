package config_test

import (
	"os"
	"testing"

	"github.com/openstatusHQ/cli/internal/config"
	"github.com/google/go-cmp/cmp"
)

var lockfile = `
"test-monitor":
    id: 1
    monitor:
      active: true
      assertions:
        - compare: eq
          kind: statusCode
          target: 200
      description: Uptime monitoring example
      frequency: 10m
      kind: http
      name: Uptime Monitor
      regions:
      - iad
      - ams
      - syd
      - jnb
      - gru
      request:
        headers:
          User-Agent: OpenStatus
        method: GET
        url: https://openstat.us
      retry: 3
`

func Test_getMonitorTrigger(t *testing.T) {
	t.Run("Read Lock file", func(t *testing.T) {
		f, err := os.CreateTemp(".", "openstatus.lock")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(f.Name()) // clean up

		if _, err := f.Write([]byte(lockfile)); err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
		out, err := config.ReadLockFile(f.Name())
		if err != nil {
			t.Fatal(err)
		}

		expect :=  &config.MonitorsLock{
			"test-monitor": {
				ID: 1,
				Monitor: config.Monitor{
					Active: true,
					Name:"Uptime Monitor",
				 	Description:"Uptime monitoring example",
					Frequency:"10m",
				 	Kind:config.HTTP,
					Retry:3,
					Public:false,
					Regions: []config.Region{ config.Iad,config.Ams,config.Syd, config.Jnb, config.Gru},
					Request: config.Request{
						URL: "https://openstat.us",
						Method: config.Get,
						Headers: map[string]string{"User-Agent": "OpenStatus"},
					},
					Assertions: []config.Assertion{
						{
							Compare: config.Eq,
							Kind: config.StatusCode,
							Target: 200,
						},
					},
				},
			},
		}

		equal := cmp.Equal(expect, out)
		if !equal {
			t.Errorf("Expected %v, got %v", expect, out)
		}
	})

	t.Run("No Lock file", func(t *testing.T) {

		out, err := config.ReadLockFile("doesnotexist")
		if err != nil {
			t.Fatal(err)
		}

		expect :=  &config.MonitorsLock{}

		equal := cmp.Equal(expect, out)
		if !equal {
			t.Errorf("Expected %v, got %v", expect, out)
		}
	})
}
