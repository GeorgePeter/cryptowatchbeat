// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import (
	"fmt"

	"github.com/elastic/beats/libbeat/cfgfile"
)

type CryptwatchConfig struct {
	Period *string `yaml:"period"`
	Cryptowatch struct {
		Markets *[]Exchange
		Periods []int64
		After   int64
	}
}
type Exchange struct {
	Name  string
	Pairs *[]string
}

type MandatoryConfigError struct {
	fieldname string
}

type yamlConfig struct {
	Cryptowatchbeat CryptwatchConfig
}

func NewCryptowatchbeatConfig() (*CryptwatchConfig, error) {
	yaml := yamlConfig{}
	err := cfgfile.Read(&yaml, "")
	if err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}
	err = yaml.Cryptowatchbeat.setDefaults()
	if err != nil {
		return nil, fmt.Errorf("Defaults could not be set: %v", err)
	}
	return &yaml.Cryptowatchbeat, nil
}

func (c *CryptwatchConfig) setDefaults() error {
	if c.Period == nil {
		*c.Period = "60s"
	}

	if c.Cryptowatch.Periods == nil {
		// default hourly
		c.Cryptowatch.Periods = []int64{3600}
	}
	if c.Cryptowatch.After == 0 {
		c.Cryptowatch.After = 1256760193
	}

	if c.Cryptowatch.Markets == nil {
		return MandatoryConfigError{"Markets"}
	}
	for _, market := range *c.Cryptowatch.Markets {
		if market.Name == "" {
			return MandatoryConfigError{"Markets[0].name"}
		}
		if len(*market.Pairs) == 0 {
			return MandatoryConfigError{"Markets[0].currency_pairs"}
		}
	}

	return nil
}

func (e MandatoryConfigError) Error() string {
	return fmt.Sprintf("Mandatory field \"%v\" was not set in config.", e.fieldname)
}
