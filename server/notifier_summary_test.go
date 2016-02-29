package main

import (
	"testing"

	"github.com/applikatoni/applikatoni/models"
)

func TestGenerateSummary(t *testing.T) {
	target := &models.Target{Name: "staging"}

	application := &models.Application{
		GitHubOwner: "shipping-co",
		GitHubRepo:  "main-web-app",
	}

	deployment := &models.Deployment{
		ApplicationName: "web",
		TargetName:      target.Name,
		Branch:          "master",
		CommitSha:       "f00b4r",
		Comment:         "hi",
	}

	user := &models.User{
		Name: "Foo Bar",
	}

	config = &Configuration{
		Host:       "example.com",
		SSLEnabled: true,
	}

	event := &DeploymentEvent{
		Deployment:  deployment,
		Application: application,
		Target:      target,
		User:        user,
	}

	expectedSuccessMsg := `main-web-app Successfully Deployed:
Foo Bar deployed master on staging :pizza:

> hi
<https://github.com/shipping-co/main-web-app/commit/f00b4r|View latest commit on GitHub>
<https://example.com/main-web-app/deployments/0|Open deployment in Applikatoni>`

	expectedFailMsg := `main-web-app Deploy Failed:
Foo Bar deployed master on staging :pizza:

> hi
<https://github.com/shipping-co/main-web-app/commit/f00b4r|View latest commit on GitHub>
<https://example.com/main-web-app/deployments/0|Open deployment in Applikatoni>`

	deployment.State = models.DEPLOYMENT_SUCCESSFUL
	actualSuccessMsg, err := generateSummary(slackTemplate, event)
	if err != nil {
		t.Errorf("generateSummary returned err: %s\n", err)
	}

	if expectedSuccessMsg != actualSuccessMsg {
		t.Errorf("sent wrong message expected=%v got=%v", expectedSuccessMsg, actualSuccessMsg)
	}

	deployment.State = models.DEPLOYMENT_FAILED
	actualFailMsg, err := generateSummary(slackTemplate, event)
	if err != nil {
		t.Errorf("generateSummary returned err: %s\n", err)
	}

	if expectedFailMsg != actualFailMsg {
		t.Errorf("sent wrong message expected=%v got=%v", expectedFailMsg, actualFailMsg)
	}
}
