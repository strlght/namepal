package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/strlght/namepal/pkg/config"
	"github.com/strlght/namepal/pkg/provider"
	"github.com/strlght/namepal/pkg/provider/docker"
	"github.com/strlght/namepal/pkg/types"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Common CommonConfig `yaml:"common"`
}

type CommonConfig struct {
	Endpoint string `yaml:"endpoint"`
}

func main() {
	watcherConfig := Config{}
	ymlConfig, err := ioutil.ReadFile("watcher.yml")
	err = yaml.Unmarshal(ymlConfig, &watcherConfig)

	var provider provider.Provider

	provider = docker.DockerProvider{}
	err = provider.Init()
	if err != nil {
		panic(err)
	}

	paramsChan := make(chan config.Params)

	go provider.Provide(paramsChan)

	registerURL := fmt.Sprintf("%s/api/register", watcherConfig.Common.Endpoint)
	for {
		params := <-paramsChan
		body := types.DnsUpdateBody{
			Data: params.Configuration.Domains,
		}
		bodyJSON, _ := json.Marshal(body)
		http.Post(registerURL, "application/json", bytes.NewBuffer(bodyJSON))
	}
}
