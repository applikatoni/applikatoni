package main

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Configuration struct {
	Host        string              `yaml:"host"`
	Application string              `yaml:"application"`
	ApiToken    string              `yaml:"api_token"`
	Stages      map[string][]string `yaml:"stages"`
}

func (c *Configuration) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("Empty `host` field in configuration file.")
	}

	if c.Application == "" {
		return fmt.Errorf("Empty `application` field in configuration file.")
	}

	if c.ApiToken == "" {
		return fmt.Errorf("Empty `api_token` field in configuration file.")
	}

	if len(c.Stages) == 0 {
		return fmt.Errorf("Empty `stages` field in configuration file.")
	}

	for k, v := range c.Stages {
		if len(v) == 0 {
			return fmt.Errorf("Empty `stages.%s` field in configuration file", k)
		}
	}

	return nil
}

func readConfiguration(path string) (*Configuration, error) {
	var config Configuration

	configFile, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
