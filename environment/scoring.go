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
		Penalty: -1,
		CheckFunction: func(_ *Environment, c *Container) bool {
			return lib.PingHost(c.Team.ContainerIP)
		},
	},
	{
		Name:    "Proxmox Status",
		Desc:    "Check if the container is running",
		Reward:  2,
		Penalty: 0,
		CheckFunction: func(e *Environment, c *Container) bool {
			ct, err := e.proxmoxAPI.GetContainer(nil, c.Team.ContainerID)

			if err != nil {
				return false
			}

			return ct.Status == "running"
		},
	},
	{
		Name:    "Nginx Status",
		Desc:    "Check if the container is running Nginx by asking the webserver for content",
		Reward:  2,
		Penalty: -2,
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
		Name:    "Team Claim",
		Desc:    "Check if the container is claimed by the expected team, if not, gives the other team 3 points",
		Reward:  1,
		Penalty: -3,
		CheckFunction: func(e *Environment, c *Container) bool {
			res, err := http.Get("http://" + c.Team.ContainerIP + "/team")
			if err != nil || res.StatusCode != 200 {
				return false
			}

			defer res.Body.Close()
			rawBody, err := io.ReadAll(res.Body)

			if err != nil {
				return false
			}

			fetchedTeam := strings.TrimSpace(string(rawBody))

			if fetchedTeam != c.Team.Name {
				for _, container := range e.Containers {
					if container.Team.Name == fetchedTeam {
						container.Team.Score += 3
					}
				}

				return false
			}

			return true
		},
	},
	{
		Name:    "Root can log in",
		Desc:    "Check if the root user can log in via SSH using the private key",
		Reward:  1,
		Penalty: -1,
		CheckFunction: func(e *Environment, c *Container) bool {
			client, err := lib.NewSSHConnectionWithRetries(c.Team.ContainerIP, 3)

			if err != nil {
				return false
			}

			defer client.Close()
			return client.Send("echo 'Hello World'") == nil
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
