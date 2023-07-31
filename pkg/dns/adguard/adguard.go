package adguard

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/strlght/namepal/pkg/types"
)


type AdguardUpdater struct {
	url   string
	token string
}

type AdguardDNSRewrite struct {
	Domain string `json:"domain"`
	Answer string `json:"answer"`
}

func (a *AdguardUpdater) Init() error {
	return nil
}

func (a *AdguardUpdater) UpdateDNSRecords(ip string, domains *[]string) error {
	entries, err := a.fetchCurrentDomains()
	if err != nil {
		log.Fatalf("error fetching current domains: %s", err)
		return err
	}

	// TODO: actually update rewrites
	log.Infof("%s", entries)
	return nil
}

func (a *AdguardUpdater) SetURL(url string) {
	a.url = url
}

func (a *AdguardUpdater) SetToken(token string) {
	a.token = token
}

func (a *AdguardUpdater) fetchCurrentDomains() (*[]types.DnsEntry, error) {
	requestURL := a.buildRequestURL()
	res, err := http.Get(requestURL)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var data *[]AdguardDNSRewrite
	err = json.Unmarshal(resBody, &data)
	if err != nil {
		return nil, err
	}
	entries := make([]types.DnsEntry, len(*data))
	for i, item := range *data {
		entry := types.DnsEntry{
			Domain: item.Answer,
			IP:     item.Answer,
		}
		entries[i] = entry
	}
	return &entries, nil
}

func (a *AdguardUpdater) buildRequestURL() string {
	return a.url + "/rewrite/list"
}

