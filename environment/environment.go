package environment

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/luthermonson/go-proxmox"
	"koth.cyber.cs.unh.edu/database"
	"koth.cyber.cs.unh.edu/lib"
)

type Container struct {
	Container                               *proxmox.Container
	Team                                    *database.DBTeam
	ServiceChecksCount, ServiceChecksPassed int
	UpdatedAt                               time.Time
}

type Environment struct {
	Containers []*Container
	proxmoxAPI *lib.ProxmoxAPI
}

func NewEnvironment(proxmoxAPI *lib.ProxmoxAPI) *Environment {
	return &Environment{
		Containers: []*Container{},
		proxmoxAPI: proxmoxAPI,
	}
}

func (e *Environment) PullFromDatabase() error {
	if len(e.Containers) > 0 {
		return fmt.Errorf("environment already populated")
	}

	teams, err := database.GetAllTeams()

	if err != nil {
		return fmt.Errorf("failed to get teams from database: %w", err)
	}

	for _, team := range teams {
		container, err := e.proxmoxAPI.GetContainer(nil, team.ContainerID)

		if err != nil {
			return fmt.Errorf("failed to get container %d: %w", team.ContainerID, err)
		}

		e.Containers = append(e.Containers, &Container{
			Container: container,
			Team:      team,
			UpdatedAt: time.Now(),
		})
	}

	if len(e.Containers) == 0 {
		lib.Log.Warning("No containers found in database")
	}

	return nil
}

func (e *Environment) createContainerStep1(teamName, ipAddress string, verbose bool) (int, error) {
	if t, _ := database.GetTeam(teamName); t != nil {
		return 0, fmt.Errorf("team %s already exists", teamName)
	}

	if verbose {
		lib.Log.Status(fmt.Sprintf("[%s][%s]: Creating container", teamName, ipAddress))
	}

	_, ctID, err := e.proxmoxAPI.CreateContainer(e.proxmoxAPI.Nodes[0], ipAddress, teamName)

	if err != nil {
		return 0, fmt.Errorf("failed to create container: %w", err)
	}

	if verbose {
		lib.Log.Success(fmt.Sprintf("[%s][%s]: Container CT-%d created", teamName, ipAddress, ctID))
	}

	return ctID, nil
}

func (e *Environment) createContainerStep2(teamName, ipAddress string, ctID int, verbose bool) error {
	if err := e.proxmoxAPI.StartContainer(nil, ctID); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	if verbose {
		lib.Log.Success(fmt.Sprintf("[%s][%s]: Container CT-%d started", teamName, ipAddress, ctID))
	}

	return nil
}

func (e *Environment) createContainerStep3(teamName, ipAddress string, ctID int, verbose bool) error {
	if err := lib.WaitOnline(ipAddress); err != nil {
		return fmt.Errorf("failed to wait for container to be online: %w", err)
	}

	if verbose {
		lib.Log.Success(fmt.Sprintf("[%s][%s]: Container CT-%d is online", teamName, ipAddress, ctID))
	}

	conn, err := lib.NewSSHConnection(ipAddress)

	if err != nil {
		return fmt.Errorf("failed to create SSH connection: %w", err)
	}

	defer conn.Close()

	if verbose {
		lib.Log.Success(fmt.Sprintf("[%s][%s]: SSH connection established", teamName, ipAddress))
	}

	var startTime time.Time

	if verbose {
		lib.Log.Status(fmt.Sprintf("[%s][%s]: Initializing container", teamName, ipAddress))
		startTime = time.Now()
	}

	if err := conn.Send("wget -qO- http://192.168.6.66/startup_script.sh | bash"); err != nil {
		return fmt.Errorf("failed to send startup script: %w", err)
	}

	if verbose {
		lib.Log.Success(fmt.Sprintf("[%s][%s]: Container initialized in %s", teamName, ipAddress, time.Since(startTime)))
	}

	return nil
}

func (e *Environment) createContainerStep4(teamName, ipAddress string, ctID int, verbose bool) error {
	if verbose {
		lib.Log.Status(fmt.Sprintf("[%s][%s]: Creating team in database", teamName, ipAddress))
	}

	team, err := database.CreateTeam(teamName, ipAddress, ctID, 0)

	if err != nil {
		return fmt.Errorf("failed to create team in database: %w", err)
	}

	if verbose {
		lib.Log.Success(fmt.Sprintf("[%s][%s]: Team created in database", teamName, ipAddress))
	}

	if verbose {
		lib.Log.Status(fmt.Sprintf("[%s][%s]: Adding container to environment", teamName, ipAddress))
	}

	container, err := e.proxmoxAPI.GetContainer(nil, ctID)

	if err != nil {
		return fmt.Errorf("failed to get container: %w", err)
	}

	e.Containers = append(e.Containers, &Container{
		Container: container,
		Team:      team,
	})

	if verbose {
		lib.Log.Success(fmt.Sprintf("[%s][%s]: Container added to environment", teamName, ipAddress))
	}

	return nil
}

func (e *Environment) CreateContainer(teamName, ipAddress string, verbose bool) (*Container, error) {
	ctID, err := e.createContainerStep1(teamName, ipAddress, verbose)

	if err != nil {
		return nil, err
	}

	if err := e.createContainerStep2(teamName, ipAddress, ctID, verbose); err != nil {
		return nil, err
	}

	if err := e.createContainerStep3(teamName, ipAddress, ctID, verbose); err != nil {
		return nil, err
	}

	if err := e.createContainerStep4(teamName, ipAddress, ctID, verbose); err != nil {
		return nil, err
	}

	return e.Containers[len(e.Containers)-1], nil
}

func (e *Environment) Print() {
	for _, container := range e.Containers {
		lib.Log.Basic(fmt.Sprintf("Container ID: %d, Team: %s, Health: %s", container.Team.ContainerID, container.Team.Name, container.Container.Status))
	}
}

func (e *Environment) JSON() ([]byte, error) {
	containers := make([]map[string]interface{}, len(e.Containers))

	for i, container := range e.Containers {
		containers[i] = map[string]interface{}{
			"container": map[string]interface{}{
				"pve_id": container.Team.ContainerID,
				"ipv4":   container.Team.ContainerIP,
				"status": container.Container.Status,
			},
			"team": map[string]interface{}{
				"name":  container.Team.Name,
				"score": container.Team.Score,
				"uptime": func() float64 {
					if container.Team.UptimeChecksTotal == 0 {
						return 1.0
					}

					return math.Round(float64(container.Team.UptimeChecksPassed)/float64(container.Team.UptimeChecksTotal)*100) / 100
				}(),
				"checks": map[string]interface{}{
					"total":  container.Team.UptimeChecksTotal,
					"passed": container.Team.UptimeChecksPassed,
					"failed": container.Team.UptimeChecksTotal - container.Team.UptimeChecksPassed,
				},
			},
			"lastUpdate": container.UpdatedAt.Format(time.RFC3339),
		}
	}

	jsonData, err := json.Marshal(containers)

	if err != nil {
		return nil, err
	}

	return jsonData, nil
}

func (e *Environment) InitAutoUpdate() chan bool {
	stop := make(chan bool)

	go func() {
		for {
			select {
			case <-time.After(30 * time.Second):
				wg := &sync.WaitGroup{}
				for _, container := range e.Containers {
					wg.Add(1)
					go func(ct *Container) {
						defer wg.Done()
						// Check proxmox status
						newCT, err := e.proxmoxAPI.GetContainer(nil, ct.Team.ContainerID)

						if err != nil {
							lib.Log.Error(fmt.Sprintf("[%s][%s]: Failed to get container status: %s", ct.Team.Name, ct.Team.ContainerIP, err.Error()))
							return
						}

						ct.Container = newCT
						ct.Team.ServiceChecksTotal = 0
						ct.Team.ServiceChecksPassed = 0

						ct.Team.UptimeChecksTotal++
						ct.Team.UptimeChecksPassed += func() int {
							if ct.Container.Status == "running" {
								ct.Team.Score += 3
								return 1
							}

							return 0
						}()

						// Check Website
						ct.Team.ServiceChecksTotal++
						ct.Team.ServiceChecksPassed += func() int {
							if res := lib.HttpGetHost(fmt.Sprintf("http://%s", ct.Team.ContainerIP)); len(res) >= 24 {
								ct.Team.Score += 5
								return 1
							}

							return 0
						}()

						// Check ping
						ct.Team.ServiceChecksTotal++
						ct.Team.ServiceChecksPassed += func() int {
							if lib.PingHost(ct.Team.ContainerIP) {
								ct.Team.Score += 2
								return 1
							}

							return 0
						}()

						ct.UpdatedAt = time.Now()
					}(container)
				}

				wg.Wait()

				for _, container := range e.Containers {
					if err := database.UpdateTeam(container.Team); err != nil {
						lib.Log.Error(fmt.Sprintf("[%s][%s]: Failed to update team in database: %s", container.Team.Name, container.Team.ContainerIP, err.Error()))
					}
				}

			case <-stop:
				return
			}
		}
	}()

	return stop
}

func (e *Environment) BulkCreate(inputs [][]string, bucketSize int) {
	var buckets [][][]string = make([][][]string, 1)

	for i, input := range inputs {
		if i%bucketSize == 0 {
			buckets = append(buckets, [][]string{})
		}

		buckets[len(buckets)-1] = append(buckets[len(buckets)-1], input)
	}

	for _, bucket := range buckets {
		wg := &sync.WaitGroup{}

		for _, input := range bucket {
			wg.Add(1)

			go func(i []string) {
				defer wg.Done()

				if _, err := e.CreateContainer(i[0], i[1], true); err != nil {
					lib.Log.Error(fmt.Sprintf("[%s][%s]: Failed to create container: %s", i[0], i[1], err.Error()))
				}
			}(input)

			time.Sleep(10 * time.Second)
		}

		wg.Wait()
	}
}

type intermediateContainer struct {
	ctID                int
	teamName, ipAddress string
}

func (e *Environment) EfficientBulkCreate(inputs [][]string, bucketSize int) {
	var buckets [][][]string = make([][][]string, 1)

	for i, input := range inputs {
		if i%bucketSize == 0 {
			buckets = append(buckets, [][]string{})
		}

		buckets[len(buckets)-1] = append(buckets[len(buckets)-1], input)
	}

	for _, bucket := range buckets {
		ctIDs := []intermediateContainer{}

		for _, input := range bucket {
			ctID, err := e.createContainerStep1(input[0], input[1], true)

			if err != nil {
				lib.Log.Error(fmt.Sprintf("[%s][%s]: Failed to create container: %s", input[0], input[1], err.Error()))
				continue
			}

			ctIDs = append(ctIDs, intermediateContainer{
				ctID:      ctID,
				teamName:  input[0],
				ipAddress: input[1],
			})
		}

		wg := &sync.WaitGroup{}

		for _, ctID := range ctIDs {
			wg.Add(1)

			go func(i intermediateContainer) {
				defer wg.Done()

				if err := e.createContainerStep2(i.teamName, i.ipAddress, i.ctID, true); err != nil {
					lib.Log.Error(fmt.Sprintf("[%s][%s]: Failed to start container: %s", i.teamName, i.ipAddress, err.Error()))
				}
			}(ctID)
		}

		wg.Wait()

		for _, ctID := range ctIDs {
			wg.Add(1)

			go func(i intermediateContainer) {
				defer wg.Done()

				if err := e.createContainerStep3(i.teamName, i.ipAddress, i.ctID, true); err != nil {
					lib.Log.Error(fmt.Sprintf("[%s][%s]: Failed to initialize container: %s", i.teamName, i.ipAddress, err.Error()))
				}
			}(ctID)
		}

		wg.Wait()

		for _, ctID := range ctIDs {
			if err := e.createContainerStep4(ctID.teamName, ctID.ipAddress, ctID.ctID, true); err != nil {
				lib.Log.Error(fmt.Sprintf("[%s][%s]: Failed to create container: %s", ctID.teamName, ctID.ipAddress, err.Error()))
			}
		}
	}
}

func (e *Environment) BulkStart(ctIDs []int, bucketSize int) {
	var buckets [][]int = make([][]int, 1)

	for i, ctID := range ctIDs {
		if i%bucketSize == 0 {
			buckets = append(buckets, []int{})
		}

		buckets[len(buckets)-1] = append(buckets[len(buckets)-1], ctID)
	}

	for _, bucket := range buckets {
		wg := &sync.WaitGroup{}

		for _, ctID := range bucket {
			wg.Add(1)

			go func(i int) {
				defer wg.Done()

				if err := e.proxmoxAPI.StartContainer(nil, i); err != nil {
					lib.Log.Error(fmt.Sprintf("Failed to start container %d: %s", i, err.Error()))
				}
			}(ctID)
		}

		wg.Wait()
	}
}

func (e *Environment) BulkStop(ctIDs []int, bucketSize int) {
	var buckets [][]int = make([][]int, 1)

	for i, ctID := range ctIDs {
		if i%bucketSize == 0 {
			buckets = append(buckets, []int{})
		}

		buckets[len(buckets)-1] = append(buckets[len(buckets)-1], ctID)
	}

	for _, bucket := range buckets {
		wg := &sync.WaitGroup{}

		for _, ctID := range bucket {
			wg.Add(1)

			go func(i int) {
				defer wg.Done()

				if err := e.proxmoxAPI.StopContainer(nil, i); err != nil {
					lib.Log.Error(fmt.Sprintf("Failed to stop container %d: %s", i, err.Error()))
				}
			}(ctID)
		}

		wg.Wait()
	}
}

func (e *Environment) BulkDelete(ctIDs []int, bucketSize int) {
	var buckets [][]int = make([][]int, 1)

	for i, ctID := range ctIDs {
		if i%bucketSize == 0 {
			buckets = append(buckets, []int{})
		}

		buckets[len(buckets)-1] = append(buckets[len(buckets)-1], ctID)
	}

	for _, bucket := range buckets {
		wg := &sync.WaitGroup{}

		for _, ctID := range bucket {
			wg.Add(1)

			go func(i int) {
				defer wg.Done()

				if err := e.proxmoxAPI.DeleteContainer(nil, i); err != nil {
					lib.Log.Error(fmt.Sprintf("Failed to delete container %d: %s", i, err.Error()))
				}
			}(ctID)
		}

		wg.Wait()
	}
}

func (e *Environment) TeamByName(name string) *Container {
	for _, container := range e.Containers {
		if container.Team.Name == name {
			return container
		}
	}

	return nil
}
