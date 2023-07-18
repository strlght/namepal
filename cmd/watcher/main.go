package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
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

func register(config Config, domains *[]string) error {
	registerURL := fmt.Sprintf("%s/api/register", config.Common.Endpoint)
	body := types.DnsUpdateBody{
		Data: *domains,
	}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return err
	}
	_, err = http.Post(registerURL, "application/json", bytes.NewBuffer(bodyJSON))
	if err != nil {
		return err
	}
	return nil
}

func main() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

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

	for {
		params := <-paramsChan
		err = register(watcherConfig, &params.Configuration.Domains)
		if err != nil {
			log.Errorf("failed to register new domains: %s", err)
		}
	}
}
