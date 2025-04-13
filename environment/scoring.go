package environment

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"koth.cyber.cs.unh.edu/lib"
)

type Check struct {
	Name          string                              `json:"name"`
	Desc          string                              `json:"desc"`
	Reward        int                                 `json:"reward"`
	Penalty       int                                 `json:"penalty"`
	CheckFunction func(*Environment, *Container) bool `json:"-"`
}

var ScoringChecks []Check = []Check{
	{
		Name:    "Ping",
		Desc:    "Check if the container is reachable",
		Reward:  3,
		Penalty: 1,
		CheckFunction: func(_ *Environment, c *Container) bool {
			return lib.PingHost(c.Team.ContainerIP)
		},
	},
	{
		Name:    "Nginx Status",
		Desc:    "Check if the container is running Nginx by asking the webserver for content",
		Reward:  2,
		Penalty: 2,
		CheckFunction: func(e *Environment, c *Container) bool {
			res, err := http.Get("http://" + c.Team.ContainerIP)

			if err != nil || res.StatusCode != 200 {
				return false
			}

			defer res.Body.Close()
			rawBody, err := io.ReadAll(res.Body)

			return err == nil && len(rawBody) >= 16
		},
	},
	{
		Name:    "Root can log in",
		Desc:    "Check if the root user can log in via SSH using the private key",
		Reward:  1,
		Penalty: 1,
		CheckFunction: func(e *Environment, c *Container) bool {
			client, err := lib.NewSSHConnectionWithRetries(c.Team.ContainerIP, 3)

			if err != nil {
				return false
			}

			defer client.Close()
			return client.Send("whoami") == nil

		},
	},
	{
		Name:    "API Availability",
		Desc:    "Query database entries from API",
		Reward:  3,
		Penalty: 1,
		CheckFunction: func(e *Environment, c *Container) bool {
			res, err := http.Get("http://" + c.Team.ContainerIP + ":5000/get-messages")

			if err != nil || res.StatusCode != 200 {
				return false
			}

			defer res.Body.Close()
			rawBody, err := io.ReadAll(res.Body)

			if err != nil {
				return false
			}

			var jsonData []any
			return json.Unmarshal(rawBody, &jsonData) == nil
		},
	},
	{
		Name:    "Prometheus",
		Desc:    "Make sure the Prometheus services are online",
		Reward:  5,
		Penalty: 5,
		CheckFunction: func(e *Environment, c *Container) bool {
			client, err := lib.NewSSHConnectionWithRetries(c.Team.ContainerIP, 3)

			if err != nil {
				return false
			}

			defer client.Close()

			if statusCode, response, err := client.SendWithOutput("systemctl status prometheus"); err != nil || statusCode != 0 || !strings.Contains(response, "active (running)") {
				return false
			}

			if statusCode, response, err := client.SendWithOutput("systemctl status node_exporter"); err != nil || statusCode != 0 || !strings.Contains(response, "active (running)") {
				return false
			}

			return true
		},
	},
	{
		Name:    "Grafana",
		Desc:    "Make sure the Grafana service is online",
		Reward:  5,
		Penalty: 1,
		CheckFunction: func(e *Environment, c *Container) bool {
			client, err := lib.NewSSHConnectionWithRetries(c.Team.ContainerIP, 3)

			if err != nil {
				return false
			}

			defer client.Close()

			if statusCode, response, err := client.SendWithOutput("systemctl status grafana"); err != nil || statusCode != 0 || !strings.Contains(response, "active (running)") {
				return false
			}

			return true
		},
	},
}

func scoringToJSON() []byte {
	bytes, err := json.Marshal(ScoringChecks)

	if err != nil {
		return []byte("[]")
	}

	return bytes
}

var ScoringJSON = scoringToJSON()
