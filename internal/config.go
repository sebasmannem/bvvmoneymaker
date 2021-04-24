package internal

import (
  "gopkg.in/yaml.v2"
  "io/ioutil"
  "os"
  "path/filepath"
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
  envConfName = "BVVCONFIG"
  defaultConfFile = "./bvvconfig.yaml"

)
type Conf struct {
  ApiKey string `yaml:"apiKey"`
  ApiSecret string `yaml:"apiSecret"`
}

func NewConfig() (config Conf, err error) {
  configfile := os.Getenv(envConfName)
  if configfile == "" {
    configfile = defaultConfFile
  }
  configfile, err = filepath.EvalSymlinks(configfile)
  if err != nil {
    return config, err
  }

  yamlConfig, err := ioutil.ReadFile(configfile)
  if err != nil {
    return config, err
  }
  err = yaml.Unmarshal(yamlConfig, &config)
  return config, err
}
