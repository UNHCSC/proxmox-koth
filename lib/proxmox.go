package lib

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/luthermonson/go-proxmox"
)

type ProxmoxAPI struct {
	client  *proxmox.Client
	bg      context.Context
	Nodes   []*proxmox.Node
	Cluster *proxmox.Cluster
}

func InitProxmox() (*ProxmoxAPI, error) {
	var api *ProxmoxAPI = &ProxmoxAPI{
		client: proxmox.NewClient(Config.Proxmox.Host, proxmox.WithHTTPClient(&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}), proxmox.WithAPIToken(Config.Proxmox.TokenID, Config.Proxmox.Secret)),
		bg:    context.Background(),
		Nodes: make([]*proxmox.Node, 0),
	}

	cluster, err := api.client.Cluster(api.bg)

	if err != nil {
		return nil, err
	}

	api.Cluster = cluster

	nodesStatuses, err := api.client.Nodes(api.bg)

	if err != nil {
		return nil, err
	}

	for _, node := range nodesStatuses {
		if node.Status == "online" {
			realNode, err := api.client.Node(api.bg, node.Node)

			if err != nil {
				return nil, err
			}

			if strings.Contains(realNode.Name, "720xd") { // Skip shitbox
				continue
			}

			api.Nodes = append(api.Nodes, realNode)
		}
	}

	return api, nil
}

func (api *ProxmoxAPI) CreateContainer(node *proxmox.Node, ipAddress, teamName string) (*proxmox.Container, int, error) {
	nextID, err := api.Cluster.NextID(api.bg)

	if err != nil {
		return nil, 0, err
	}

	ctJob, err := node.NewContainer(api.bg, nextID, proxmox.ContainerOption{
		Name:  "ostemplate",
		Value: Config.Container.Template,
	}, proxmox.ContainerOption{
		Name:  "storage",
		Value: Config.Container.StoragePool,
	}, proxmox.ContainerOption{
		Name:  "hostname",
		Value: strings.ToLower(strings.ReplaceAll(fmt.Sprintf("%s-%s", Config.Container.HostnamePrefix, teamName), " ", "-")),
	}, proxmox.ContainerOption{
		Name:  "password",
		Value: "password",
	}, proxmox.ContainerOption{
		Name:  "rootfs",
		Value: fmt.Sprintf("volume=%s:%d", Config.Container.StoragePool, Config.Container.StorageGB),
	}, proxmox.ContainerOption{
		Name:  "memory",
		Value: Config.Container.MemoryMB,
	}, proxmox.ContainerOption{
		Name:  "cores",
		Value: Config.Container.Cores,
	}, proxmox.ContainerOption{
		Name:  "net0",
		Value: fmt.Sprintf("name=eth0,bridge=vmbr0,firewall=1,gw=%s,ip=%s/%d", Config.Container.GatewayIPv4, ipAddress, Config.Container.IndividualCIDR),
	}, proxmox.ContainerOption{
		Name:  "nameserver",
		Value: Config.Container.Nameserver,
	}, proxmox.ContainerOption{
		Name:  "searchdomain",
		Value: Config.Container.SearchDomain,
	}, proxmox.ContainerOption{
		Name:  "unprivileged",
		Value: true,
	}, proxmox.ContainerOption{
		Name:  "features",
		Value: "nesting=1",
	}, proxmox.ContainerOption{
		Name:  "ssh-public-keys",
		Value: SSHPublicKey,
	})

	if err != nil {
		return nil, 0, err
	}

	if err := ctJob.Wait(api.bg, time.Second, time.Minute*3); err != nil {
		return nil, 0, err
	}

	ct, err := node.Container(api.bg, nextID)

	if err != nil {
		return nil, 0, err
	}

	return ct, nextID, nil
}

func (api *ProxmoxAPI) NodeForContainer(containerID int) (*proxmox.Node, error) {
	for _, node := range api.Nodes {
		_, err := node.Container(api.bg, containerID)

		if err == nil {
			return node, nil
		}
	}

	return nil, fmt.Errorf("container %d not found on any node", containerID)
}

func (api *ProxmoxAPI) StartContainer(node *proxmox.Node, containerID int) error {
	var err error
	if node == nil {
		node, err = api.NodeForContainer(containerID)

		if err != nil {
			return err
		}
	}

	ct, err := node.Container(api.bg, containerID)

	if err != nil {
		return err
	}

	task, err := ct.Start(api.bg)

	if err != nil {
		return err
	}

	if err := task.Wait(api.bg, time.Second, time.Minute*3); err != nil {
		return err
	}

	return nil
}

func (api *ProxmoxAPI) StopContainer(node *proxmox.Node, containerID int) error {
	var err error
	if node == nil {
		node, err = api.NodeForContainer(containerID)

		if err != nil {
			return err
		}
	}

	ct, err := node.Container(api.bg, containerID)

	if err != nil {
		return err
	}

	task, err := ct.Stop(api.bg)

	if err != nil {
		return err
	}

	if err := task.Wait(api.bg, time.Second, time.Minute*3); err != nil {
		return err
	}

	return nil
}

func (api *ProxmoxAPI) DeleteContainer(node *proxmox.Node, containerID int) error {
	var err error
	if node == nil {
		node, err = api.NodeForContainer(containerID)

		if err != nil {
			return err
		}
	}

	ct, err := node.Container(api.bg, containerID)

	if err != nil {
		return err
	}

	task, err := ct.Delete(api.bg)

	if err != nil {
		return err
	}

	if err := task.Wait(api.bg, time.Second, time.Minute*3); err != nil {
		return err
	}

	return nil
}

func (api *ProxmoxAPI) GetContainer(node *proxmox.Node, containerID int) (*proxmox.Container, error) {
	var err error
	if node == nil {
		node, err = api.NodeForContainer(containerID)

		if err != nil {
			return nil, err
		}
	}

	ct, err := node.Container(api.bg, containerID)

	if err != nil {
		return nil, err
	}

	return ct, nil
}

func (api *ProxmoxAPI) RelevantContainers() ([]*proxmox.Container, error) {
	containers := make([]*proxmox.Container, 0)

	for _, node := range api.Nodes {
		nodeContainers, err := node.Containers(api.bg)

		if err != nil {
			return nil, err
		}

		for _, container := range nodeContainers {
			if strings.Index(container.Name, Config.Container.HostnamePrefix) == 0 {
				containers = append(containers, container)
			}
		}
	}

	return containers, nil
}

func (api *ProxmoxAPI) BulkStart(ctIDs []int, bucketSize int) {
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

				if err := api.StartContainer(nil, i); err != nil {
					Log.Error(fmt.Sprintf("Failed to start container %d: %s", i, err.Error()))
				}
			}(ctID)
		}

		wg.Wait()
	}
}

func (api *ProxmoxAPI) BulkStop(ctIDs []int, bucketSize int) {
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

				if err := api.StopContainer(nil, i); err != nil {
					Log.Error(fmt.Sprintf("Failed to stop container %d: %s", i, err.Error()))
				}
			}(ctID)
		}

		wg.Wait()
	}
}

func (api *ProxmoxAPI) BulkDelete(ctIDs []int, bucketSize int) {
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

				if err := api.DeleteContainer(nil, i); err != nil {
					Log.Error(fmt.Sprintf("Failed to delete container %d: %s", i, err.Error()))
				}
			}(ctID)
		}

		wg.Wait()
	}
}
