package provider

import (
	"github.com/strlght/namepal/pkg/config"
)

type Provider interface {
	Init() error
	Provide(paramsChan chan<- config.Params) error
}
