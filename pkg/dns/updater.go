package dns

type Updater interface {
	Init() error
	UpdateDNSRecords(ip string, domains *[]string) error
}
