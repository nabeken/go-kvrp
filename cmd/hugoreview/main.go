package main

import (
	"log"
	"net/http"

	"github.com/fsouza/go-dockerclient"

	"github.com/nabeken/hugoreview"
)

func main() {
	d, err := docker.NewClient(hugoreview.Getenv("DOCKER_HOST", "unix:///var/run/docker.sock"))
	if err != nil {
		log.Fatal(err)
	}

	redis := hugoreview.NewRedisStore()

	containerHandler := &hugoreview.ContainerHandler{
		Client: d,
		Store:  redis,
		Image:  "nabeken/docker-hugo-server",
	}

	pullRequestHandler := &hugoreview.PullRequestHandler{
		ContainerHandler: containerHandler,
		Domain:           hugoreview.Getenv("HUGOREVIEW_DOMAIN", "hugo-local.mahiru.moe"),
		Port:             hugoreview.Getenv("PORT", "8000"),
	}

	proxyHandler := hugoreview.NewHandler(redis)

	http.Handle("/_webhooks", pullRequestHandler)
	http.Handle("/", proxyHandler)
	log.Print("Starting Redis-based reverse proxy...")
	log.Fatal(http.ListenAndServe(hugoreview.Addr(), nil))
}
