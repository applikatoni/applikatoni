package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/applikatoni/applikatoni/models"
)

func TestNotifyWebhooks(t *testing.T) {
	target := &models.Target{Name: "staging", BugsnagApiKey: "APIKEY"}
	application := &models.Application{
		GitHubOwner: "shipping-co",
		GitHubRepo:  "main-web-app",
	}

	user := buildUser(1234, "Bobby")
	deployment := buildDeployment(user.Id)
	deployment.User = user

	event := &DeploymentEvent{
		State:       models.DEPLOYMENT_SUCCESSFUL,
		Deployment:  deployment,
		Application: application,
		Target:      target,
		User:        user,
	}

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		msg := &WebhookMsg{}
		err := json.NewDecoder(r.Body).Decode(msg)
		if err != nil {
			t.Errorf("decoding failed: %s", err)
		}
		if msg.State != event.State {
			t.Errorf("wrong message state. got=%s", msg.State)
		}
	}

	firstWebhook := httptest.NewServer(http.HandlerFunc(testHandler))
	defer firstWebhook.Close()
	secondWebhook := httptest.NewServer(http.HandlerFunc(testHandler))
	defer secondWebhook.Close()

	target.Webhooks = []string{firstWebhook.URL, secondWebhook.URL}

	NotifyWebhooks(event)
}
