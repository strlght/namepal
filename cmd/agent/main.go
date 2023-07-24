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

func register(config *Config, domains *[]string) error {
	if len(*domains) == 0 {
		log.Debug("received empty domains list")
		return nil
	}

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

func parseConfig() (*Config, error) {
	agentConfig := Config{}
	ymlConfig, err := ioutil.ReadFile("agent.yml")
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(ymlConfig, &agentConfig)
	if err != nil {
		return nil, err
	}
	return &agentConfig, nil
}

func createProvider(config *Config) (provider.Provider, error) {
	provider := docker.DockerProvider{}
	err := provider.Init()
	if err != nil {
		return nil, err
	}
	return provider, nil
}

func loop(agentConfig *Config, provider provider.Provider) {
	paramsChan := make(chan config.Params)

	go provider.Provide(paramsChan)

	for {
		params := <-paramsChan
		err := register(agentConfig, &params.Configuration.Domains)
		if err != nil {
			log.Errorf("failed to register new domains: %s", err)
		}
	}
}

func init() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

func main() {
	agentConfig, err := parseConfig()
	if err != nil {
		log.Fatalf("failed to process config: %s", err)
		os.Exit(1)
	}

	provider, err := createProvider(agentConfig)
	if err != nil {
		log.Fatalf("failed to create provider: %s", err)
		os.Exit(1)
	}

	loop(agentConfig, provider)
}
