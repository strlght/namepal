package docker

import (
	"context"
	"regexp"
	"strings"

	"github.com/strlght/namepal/pkg/config"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

type DockerProvider struct {
	client *client.Client
}

func (p DockerProvider) Init() error {
	var err error
	p.client, err = client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}
	return nil
}

func (p DockerProvider) Provide(paramsChan chan<- config.Params) error {
	ctx := context.Background()
	p.client.NegotiateAPIVersion(ctx)

	labelRegexp := regexp.MustCompile("traefik.http.routers.\\w+.rule")
	hostRegexp := regexp.MustCompile("Host\\(`(.+)`\\)")

	f := filters.NewArgs()
	f.Add("type", "container")
	options := dockertypes.EventsOptions{
		Filters: f,
	}

	startStopHandle := func() {
		containers, err := p.client.ContainerList(ctx, dockertypes.ContainerListOptions{})
		if err != nil {
			panic(err)
		}

		domains := make([]string, 0)
		for _, container := range containers {
			id := container.ID[:10]
			result, err := p.client.ContainerInspect(ctx, id)
			if err != nil {
				panic(err)
			}

			for key, value := range result.Config.Labels {
				if labelRegexp.MatchString(key) && hostRegexp.MatchString(value) {
					log.Debugf("found matching container: %s", id)
					subMatches := hostRegexp.FindStringSubmatch(value)
					domains = append(domains, subMatches[1])
				}
			}

			params := config.Params{
				ProviderName:  "docker",
				Configuration: &config.Configuration{Domains: domains},
			}
			paramsChan <- params
		}
	}
	startStopHandle()

	eventsc, errc := p.client.Events(ctx, options)
	for {
		select {
		case event := <-eventsc:
			if event.Action == "start" ||
				event.Action == "die" ||
				strings.HasPrefix(event.Action, "health_status") {
				startStopHandle()
			}
		case err := <-errc:
			return err
		case <-ctx.Done():
			return nil
		}
	}
}
