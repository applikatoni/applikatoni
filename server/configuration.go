package main

import (
	"encoding/json"
	"io/ioutil"

	"github.com/applikatoni/applikatoni/models"
)

type Configuration struct {
	Host               string                `json:"host"`
	SSLEnabled         bool                  `json:"ssl_enabled"`
	SessionSecret      string                `json:"session_secret"`
	Oauth2StateString  string                `json:"oauth2_state_string"`
	GitHubClientId     string                `json:"github_client_id"`
	GitHubClientSecret string                `json:"github_client_secret"`
	MandrillAPIKey     string                `json:"mandrill_api_key"`
	MailgunBaseURL     string                `json:"mailgun_base_url"`
	MailgunAPIKey      string                `json:"mailgun_api_key"`
	Applications       []*models.Application `json:"applications"`
}

func readConfiguration(path string) (*Configuration, error) {
	var config Configuration

	configFile, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(configFile, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
