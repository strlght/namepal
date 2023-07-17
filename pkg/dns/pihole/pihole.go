package pihole

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	log "github.com/sirupsen/logrus"
	"github.com/strlght/namepal/pkg/types"
)

type PiholeUpdater struct {
	url   string
	token string
}

type PiholeDNSListResponse struct {
	Data [][]string `json:"data"`
}

func (p *PiholeUpdater) Init() error {
	return nil
}

func (p *PiholeUpdater) UpdateDNSRecords(ip string, domains *[]string) error {
	entries, err := p.fetchCurrentDomains()
	if err != nil {
		log.Fatalf("error fetching current domains: %s\n", err)
		return err
	}

	err = p.removeOutdatedDomains(ip, entries, domains)
	if err != nil {
		log.Fatalf("error removing outdated domains: %s\n", err)
		return err
	}

	err = p.submitNewDomains(ip, domains)
	if err != nil {
		log.Fatalf("error submitting new domains: %s\n", err)
		return err
	}
	return nil
}

func (p *PiholeUpdater) SetURL(url string) {
	p.url = url
}

func (p *PiholeUpdater) SetToken(token string) {
	p.token = token
}

func (p *PiholeUpdater) removeOutdatedDomains(ip string, currentEntries *[]types.DnsEntry, requestedDomains *[]string) error {
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
			deleteURL := p.buildDeleteURL(entry.Domain, entry.IP)
			_, err := http.Get(deleteURL)
			if err != nil {
				log.Infof("failed to delete outdated entry: %s", err)
				return err
			}
		}
	}
	return nil
}

func (p *PiholeUpdater) submitNewDomains(ip string, domains *[]string) error {
	for _, domain := range *domains {
		log.Infof("adding new entry: %s %s", ip, domain)
		addURL := p.buildAddURL(domain, ip)
		_, err := http.Get(addURL)
		if err != nil {
			log.Infof("failed to delete outdated entry: %s", err)
			return err
		}
	}
	return nil
}

func (p *PiholeUpdater) fetchCurrentDomains() (*[]types.DnsEntry, error) {
	requestURL := p.buildRequestURL()
	res, err := http.Get(requestURL)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var data *PiholeDNSListResponse
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

func (p *PiholeUpdater) buildPiholeUrl(modifier func(*url.Values)) string {
	baseURL := p.url
	v := url.Values{}
	v.Set("auth", p.token)
	v.Set("customdns", "")
	modifier(&v)
	return baseURL + "?" + v.Encode()
}

func (p *PiholeUpdater) buildRequestURL() string {
	return p.buildPiholeUrl(func(v *url.Values) {
		v.Set("action", "get")
	})
}

func (p *PiholeUpdater) buildAddURL(domain string, IP string) string {
	return p.buildPiholeUrl(func(v *url.Values) {
		v.Set("action", "add")
		v.Set("domain", domain)
		v.Set("ip", IP)
	})
}

func (p *PiholeUpdater) buildDeleteURL(domain string, IP string) string {
	return p.buildPiholeUrl(func(v *url.Values) {
		v.Set("action", "delete")
		v.Set("domain", domain)
		v.Set("ip", IP)
	})
}
