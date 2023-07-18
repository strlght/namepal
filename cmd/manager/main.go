package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/strlght/namepal/pkg/dns"
	"github.com/strlght/namepal/pkg/dns/pihole"
	"github.com/strlght/namepal/pkg/types"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Common *CommonConfig `yaml:"common"`
	Pihole *PiholeConfig `yaml:"pihole"`
}

type CommonConfig struct {
}

type PiholeConfig struct {
	URL   string `yaml:"url"`
	Token string `yaml:"token"`
}

func ParseRequestBody(r *http.Request) (*types.DnsUpdateBody, error) {
	bodyRaw, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	var body *types.DnsUpdateBody
	err = json.Unmarshal(bodyRaw, &body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func ExtractIP(r *http.Request) string {
	forwardedFor := r.Header["X-Forwarded-For"]
	if forwardedFor != nil && len(forwardedFor) == 1 {
		return forwardedFor[0]
	} else {
		return strings.Split(r.RemoteAddr, ":")[0]
	}
}

func main() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	var config Config
	ymlConfig, err := ioutil.ReadFile("manager.yml")
	err = yaml.Unmarshal(ymlConfig, &config)

	var updater dns.Updater

	if config.Pihole != nil {
		piholeUpdater := pihole.PiholeUpdater{}
		updater = &piholeUpdater

		piholeUpdater.SetToken(config.Pihole.Token)
		piholeUpdater.SetURL(config.Pihole.URL)

		err = piholeUpdater.Init()
		if err != nil {
			log.Fatalf("failed to initialize pihole updater: %s", err)
			os.Exit(1)
		}
	} else {
		log.Fatal("updater should be defined in config")
		os.Exit(1)
	}

	http.HandleFunc("/api/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			return
		}

		body, err := ParseRequestBody(r)
		if err != nil {
			log.Fatalf("error parsing request body: %s", err)
			return
		}
		ip := ExtractIP(r)

		err = updater.UpdateDNSRecords(ip, &body.Data)
		if err != nil {
			log.Fatalf("failed updating DNS records: %s", err)
			return
		}
	})

	err = http.ListenAndServe(":8000", nil)
	if errors.Is(err, http.ErrServerClosed) {
		log.Info("server closed")
	} else if err != nil {
		log.Fatalf("error starting server: %s", err)
		os.Exit(1)
	}
}
