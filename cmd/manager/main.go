package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/strlght/namepal/pkg/types"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type DnsListResponse struct {
	Data [][]string `json:"data"`
}

type Config struct {
	Common CommonConfig `yaml:"common"`
	Pihole PiholeConfig `yaml:"pihole"`
}

type CommonConfig struct {
}

type PiholeConfig struct {
	URL   string `yaml:"url"`
	Token string `yaml:"token"`
}

var config = Config{}

func Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		return
	}

	body, err := ParseRequestBody(r)
	if err != nil {
		log.Fatalf("error parsing request body: %s\n", err)
		return
	}
	ip := ExtractIP(r)
	entries, err := FetchCurrentDomains()
	if err != nil {
		log.Fatalf("error fetching current domains: %s\n", err)
		return
	}

	err = RemoveOutdatedDomains(ip, entries, &body.Data)
	if err != nil {
		log.Fatalf("error removing outdated domains: %s\n", err)
		return
	}

	err = SubmitNewDomains(ip, &body.Data)
	if err != nil {
		log.Fatalf("error submitting new domains: %s\n", err)
		return
	}
}

func RemoveOutdatedDomains(ip string, currentEntries *[]types.DnsEntry, requestedDomains *[]string) error {
	for _, entry := range *currentEntries {
		found := false
		for _, domain := range *requestedDomains {
			if domain == entry.Domain {
				found = true
				break
			}
		}

		usesCurrentIP := entry.IP == ip
		shouldDelete := (usesCurrentIP && !found) || (!usesCurrentIP && found)
		if shouldDelete {
			log.Infof("deleting outdated entry: %s %s", entry.IP, entry.Domain)
			deleteURL := BuildDeleteURL(config.Pihole, entry.Domain, entry.IP)
			_, err := http.Get(deleteURL)
			if err != nil {
				log.Infof("failed to delete outdated entry: %s", err)
				return err
			}
		}
	}
	return nil
}

func SubmitNewDomains(ip string, domains *[]string) error {
	for _, domain := range *domains {
		log.Infof("adding new entry: %s %s", ip, domain)
		addURL := BuildAddURL(config.Pihole, domain, ip)
		_, err := http.Get(addURL)
		if err != nil {
			log.Infof("failed to delete outdated entry: %s", err)
			return err
		}
	}
	return nil
}

func FetchCurrentDomains() (*[]types.DnsEntry, error) {
	requestURL := BuildRequestURL(config.Pihole)
	res, err := http.Get(requestURL)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var data *DnsListResponse
	err = json.Unmarshal(resBody, &data)
	if err != nil {
		return nil, err
	}
	entries := make([]types.DnsEntry, len(data.Data))
	for i, response := range data.Data {
		entry := types.DnsEntry{
			Domain: response[0],
			IP:     response[1],
		}
		entries[i] = entry
	}
	return &entries, nil
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

func BuildPiholeUrl(config PiholeConfig, modifier func(*url.Values)) string {
	baseURL := config.URL
	v := url.Values{}
	v.Set("auth", config.Token)
	v.Set("customdns", "")
	modifier(&v)
	return baseURL + "?" + v.Encode()
}

func BuildRequestURL(config PiholeConfig) string {
	return BuildPiholeUrl(config, func(v *url.Values) {
		v.Set("action", "get")
	})
}

func BuildAddURL(config PiholeConfig, domain string, IP string) string {
	return BuildPiholeUrl(config, func(v *url.Values) {
		v.Set("action", "add")
		v.Set("domain", domain)
		v.Set("ip", IP)
	})
}

func BuildDeleteURL(config PiholeConfig, domain string, IP string) string {
	return BuildPiholeUrl(config, func(v *url.Values) {
		v.Set("action", "delete")
		v.Set("domain", domain)
		v.Set("ip", IP)
	})
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

	ymlConfig, err := ioutil.ReadFile("manager.yml")
	err = yaml.Unmarshal(ymlConfig, &config)

	http.HandleFunc("/api/register", Register)
	err = http.ListenAndServe(":8000", nil)
	if errors.Is(err, http.ErrServerClosed) {
		log.Info("server closed\n")
	} else if err != nil {
		log.Fatalf("error starting server: %s\n", err)
		os.Exit(1)
	}
}
