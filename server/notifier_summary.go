package main

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/applikatoni/applikatoni/models"
)

func generateSummary(t *template.Template, ev *DeploymentEvent) (string, error) {
	var scheme string
	if config.SSLEnabled {
		scheme = "https"
	} else {
		scheme = "http"
	}

	var success bool
	if ev.State == models.DEPLOYMENT_SUCCESSFUL {
		success = true
	} else {
		success = false
	}

	deploymentUrl := fmt.Sprintf("%s://%s/%v/deployments/%v", scheme, config.Host,
		ev.Application.GitHubRepo, ev.Deployment.Id)

	gitHubUrl := fmt.Sprintf("https://github.com/%v/%v/commit/%v",
		ev.Application.GitHubOwner, ev.Application.GitHubRepo,
		ev.Deployment.CommitSha)

	var summary bytes.Buffer
	err := t.Execute(&summary, map[string]interface{}{
		"GitHubRepo":    ev.Application.GitHubRepo,
		"Success":       success,
		"Branch":        ev.Deployment.Branch,
		"Target":        ev.Deployment.TargetName,
		"Username":      ev.User.Name,
		"Comment":       ev.Deployment.Comment,
		"GitHubUrl":     gitHubUrl,
		"DeploymentURL": deploymentUrl,
	})

	return summary.String(), err
}
