package internal

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

/*
 * This is an example utilising all functions of the GO Bitvavo API wrapper.
 * The APIKEY and APISECRET should be replaced by your own key and secret.
 * For public functions the APIKEY and SECRET can be removed.
 * Documentation: https://docs.bitvavo.com
 * Bitvavo: https://bitvavo.com
 * README: https://github.com/bitvavo/go-bitvavo-api
 */

const (
	envConfName     = "BVVCONFIG"
	defaultConfFile = "./bvvconfig.yaml"
	defaultCurrency = "EUR"
)

type bvvApiConfig struct {
	Key    string `yaml:"key"`
	Secret string `yaml:"secret"`
	Debug  bool   `yaml:"debug""`
}

type bvvMarketConfig struct {
	// When more then this level of currency is available, we can sell
	MinLevel string `yaml:"min"`
	MaxLevel string `yaml:"max"`
}

type bvvConfig struct {
	Api             bvvApiConfig               `yaml:"api"`
	DefaultCurrency string                     `yaml:"defaultCurrency"`
	Markets         map[string]bvvMarketConfig `yaml:"markets"`
}

func NewConfig() (config bvvConfig, err error) {
	configFile := os.Getenv(envConfName)
	if configFile == "" {
		configFile = defaultConfFile
	}
	configFile, err = filepath.EvalSymlinks(configFile)
	if err != nil {
		return config, err
	}

	yamlConfig, err := ioutil.ReadFile(configFile)
	if err != nil {
		return config, err
	}
	err = yaml.Unmarshal(yamlConfig, &config)
	if config.DefaultCurrency == "" {
		config.DefaultCurrency = defaultCurrency
	}
	return config, err
}