package config

import (
	"github.com/Kaese72/sdup-lib/httpsdup"
	sdupclientconfig "github.com/Kaese72/sdup-lib/sdupclient/config"
)

type Config struct {
	SDUPClientConfig sdupclientconfig.Config `json:"sdup-client"`
	SDUPServerConfig httpsdup.Config         `json:"sdup-server"`
}

func (conf *Config) PopulateExample() {
	conf.SDUPClientConfig = sdupclientconfig.Config{}
	conf.SDUPClientConfig.PopulateExample()

	conf.SDUPServerConfig = httpsdup.Config{}
	conf.SDUPServerConfig.PopulateExample()
}

func (conf Config) Validate() error {
	if err := conf.SDUPClientConfig.Validate(); err != nil {
		return err
	}
	if err := conf.SDUPServerConfig.Validate(); err != nil {
		return err
	}
	return nil
}
