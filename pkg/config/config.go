package config

type Params struct {
	ProviderName  string
	Configuration *Configuration
}

type Configuration struct {
	Domains []string
}
