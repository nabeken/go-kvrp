package hugoreview

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/fsouza/go-dockerclient"
)

type Repository struct {
	CloneURL string `json:"clone_url"`
}

type PullRequestEvent struct {
	Action string `json:"action"`
	Number int    `json:"number"`

	Repository Repository `json:"repository"`
}

type PullRequestHandler struct {
	Secret string
	Domain string
	Port   string

	ContainerHandler *ContainerHandler
}

func (h *PullRequestHandler) formatHost(number int) string {
	return fmt.Sprintf("pr-%d.%s", number, h.Domain)
}

func (h *PullRequestHandler) close(payload *PullRequestEvent) error {
	host := h.formatHost(payload.Number)
	httphost := host + ":" + h.Port
	c := h.ContainerHandler.Store.GetHost(httphost)
	err := h.ContainerHandler.Destroy(c)
	if err != nil {
		return err
	}
	return h.ContainerHandler.Deregister(httphost)
}

func (h *PullRequestHandler) open(payload *PullRequestEvent) error {
	env := []string{
		fmt.Sprintf("GIT_REPO=%s", payload.Repository.CloneURL),
		fmt.Sprintf("GITHUB_PR_NUMBER=%d", payload.Number),
	}
	host := h.formatHost(payload.Number)
	c, err := h.ContainerHandler.Run(host, h.Port, env)
	if err != nil {
		return err
	}
	return h.ContainerHandler.Register(host+":"+h.Port, c)
}

func (h *PullRequestHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if req.Header.Get("X-Github-Event") != "ping" {
		log.Print("Ping")
		io.Copy(rw, req.Body)
		return
	}

	if req.Header.Get("X-Github-Event") != "pull_request" {
		http.Error(rw, "Bad Request", http.StatusBadRequest)
		return
	}

	payload := &PullRequestEvent{}
	defer req.Body.Close()
	if err := json.NewDecoder(req.Body).Decode(payload); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	log.Print(*payload)

	switch payload.Action {
	case "opened", "reopened":
		if err := h.open(payload); err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
	case "synchronize":
		// destroy the container
		// launch a new container
		// register its IPAddr:Port to redis
	case "closed":
		// destroy the container
		if err := h.close(payload); err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
	default:
		http.Error(rw, "Bad Request", http.StatusBadRequest)
		return
	}
}

// ContainerHandler manages a container by key
type ContainerHandler struct {
	*docker.Client

	Store Store
	Image string
}

// Run creates new container and starts the container.
func (h *ContainerHandler) Run(host, port string, env []string) (*Container, error) {
	env = append(env, "PORT="+port)
	opts := docker.CreateContainerOptions{
		Config: &docker.Config{
			Image:    h.Image,
			Hostname: host,
			Env:      env,
		},
	}

	c, err := h.CreateContainer(opts)
	if err != nil {
		return nil, err
	}

	if err := h.StartContainer(c.ID, &docker.HostConfig{}); err != nil {
		return nil, err
	}

	c, err = h.InspectContainer(c.ID)
	if err != nil {
		return nil, err
	}

	return &Container{
		ID:   c.ID,
		Host: c.NetworkSettings.IPAddress + ":" + port,
	}, nil
}

// Destroy destroys a container pointed by c
func (h *ContainerHandler) Destroy(c *Container) error {
	opts := docker.RemoveContainerOptions{
		ID:            c.ID,
		RemoveVolumes: true,
		Force:         true,
	}
	return h.RemoveContainer(opts)
}

// Register registers key with c to Store
func (h *ContainerHandler) Register(host string, c *Container) error {
	return h.Store.SetHost(host, c)
}

// Deregister removes key from Store
func (h *ContainerHandler) Deregister(key string) error {
	return h.Store.DeleteHost(key)
}
