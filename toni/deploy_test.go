package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestBuildDeploymentURL(t *testing.T) {
	tests := []struct {
		hostURL         string
		applicationName string
		expected        string
	}{
		{"http://toni.company.com", "foobar", "http://toni.company.com/foobar/deployments"},
		{"http://localhost", "herpderp", "http://localhost/herpderp/deployments"},
		{"https://localhost", "herpderp", "https://localhost/herpderp/deployments"},
		{"https://localhost/", "herpderp", "https://localhost/herpderp/deployments"},
		{"https://localhost/wrong", "herpderp", "https://localhost/herpderp/deployments"},
		{"https://localhost/", "/herpderp/", "https://localhost/herpderp/deployments"},
	}

	for _, test := range tests {
		url, err := buildDeploymentURL(test.hostURL, test.applicationName)
		if err != nil {
			t.Error(err)
		}
		if url.String() != test.expected {
			t.Errorf("expected=%q, got=%q", test.expected, url)
		}
	}
}

func TestBuildDeploymentLogURL(t *testing.T) {
	tests := []struct {
		hostURL        string
		deploymentPath string
		expected       string
	}{
		{"http://toni.company.com", "/foobar/deployments/999", "ws://toni.company.com/foobar/deployments/999/log"},
		{"http://toni.company.com", "/foobar/deployments/9/", "ws://toni.company.com/foobar/deployments/9/log"},
		{"https://toni.company.com", "/foobar/deployments/9/", "wss://toni.company.com/foobar/deployments/9/log"},
		{"https://toni.company.com/", "/foobar/deployments/9/", "wss://toni.company.com/foobar/deployments/9/log"},
		{"https://sub.toni.company.com/", "/foobar/deployments/9/", "wss://sub.toni.company.com/foobar/deployments/9/log"},
	}

	for _, test := range tests {
		url, err := buildDeploymentLogURL(test.hostURL, test.deploymentPath)
		if err != nil {
			t.Error(err)
		}
		if url.String() != test.expected {
			t.Errorf("expected=%q, got=%q", test.expected, url)
		}
	}
}

func TestCreateDeployment(t *testing.T) {
	data := url.Values{}
	data.Set("target", "staging-noop")
	data.Set("commitsha", "2df09e2cb924fdae62ec42a2e8ff2a0bc3f10175")
	data.Set("branch", "develop")
	data.Set("comment", "this is a comment")
	data.Add("stages[]", "CODE_DEPLOYMENT")

	config := &Configuration{
		Application: "foobar",
		ApiToken:    "TOKEN",
		Host:        "http://localhost:8080",
	}

	tests := []struct {
		testServerHandler func(http.ResponseWriter, *http.Request)
		expectedLocation  string
		expectedError     error
	}{
		{
			func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "/foobar/deployments/999", http.StatusSeeOther)
			},
			"/foobar/deployments/999",
			nil,
		},
		{
			func(w http.ResponseWriter, r *http.Request) {
				http.NotFound(w, r)
			},
			"",
			UnexpectedResponse{404, "404 page not found\n"},
		},
		{
			func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "invalid commit sha", 422)
			},
			"",
			UnexpectedResponse{422, "invalid commit sha\n"},
		},
		{
			func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "database is broken", http.StatusInternalServerError)
			},
			"",
			UnexpectedResponse{500, "database is broken\n"},
		},
	}

	for _, test := range tests {
		ts := httptest.NewServer(http.HandlerFunc(test.testServerHandler))
		defer ts.Close()

		u, err := url.Parse(ts.URL)
		if err != nil {
			t.Error(err)
		}

		deploymentLocation, err := createDeployment(config, u, data)
		if err != test.expectedError {
			t.Errorf("expected createDeploymentUrl to return %q as error. got=%q", test.expectedError, err)
		}

		if deploymentLocation != test.expectedLocation {
			t.Errorf("deploymentLocation wrong. got=%s, expected=%s",
				deploymentLocation, test.expectedLocation)
		}
	}
}
