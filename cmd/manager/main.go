package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/strlght/namepal/pkg/types"

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
		fmt.Printf("error parsing request body: %s\n", err)
		return
	}
	ip := ExtractIP(r)
	entries, err := FetchCurrentDomains()
	if err != nil {
		fmt.Printf("error fetching current domains: %s\n", err)
		return
	}

	err = RemoveOutdatedDomains(ip, entries, &body.Data)
	if err != nil {
		fmt.Printf("error removing outdated domains: %s\n", err)
		return
	}

	err = SubmitNewDomains(ip, &body.Data)
	if err != nil {
		fmt.Printf("error submitting new domains: %s\n", err)
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
			deleteURL := BuildDeleteURL(config.Pihole, entry.Domain, entry.IP)
			_, err := http.Get(deleteURL)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func SubmitNewDomains(ip string, domains *[]string) error {
	for _, domain := range *domains {
		addURL := BuildAddURL(config.Pihole, domain, ip)
		_, err := http.Get(addURL)
		if err != nil {
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

func BuildRequestURL(config PiholeConfig) string {
	baseURL := config.URL
	v := url.Values{}
	v.Set("auth", config.Token)
	v.Set("action", "get")
	v.Set("customdns", "")
	return baseURL + "?" + v.Encode()
}

func BuildAddURL(config PiholeConfig, domain string, IP string) string {
	baseURL := config.URL
	v := url.Values{}
	v.Set("auth", config.Token)
	v.Set("action", "add")
	v.Set("customdns", "")
	v.Set("domain", domain)
	v.Set("ip", IP)
	return baseURL + "?" + v.Encode()
}

func BuildDeleteURL(config PiholeConfig, domain string, IP string) string {
	baseURL := config.URL
	v := url.Values{}
	v.Set("auth", config.Token)
	v.Set("action", "delete")
	v.Set("customdns", "")
	v.Set("domain", domain)
	v.Set("ip", IP)
	return baseURL + "?" + v.Encode()
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
	ymlConfig, err := ioutil.ReadFile("manager.yml")
	err = yaml.Unmarshal(ymlConfig, &config)

	http.HandleFunc("/api/register", Register)
	err = http.ListenAndServe(":8000", nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}
