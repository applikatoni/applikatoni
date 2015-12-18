package main

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/applikatoni/applikatoni/deploy"
	"github.com/applikatoni/applikatoni/models"
)

func generateSummary(t *template.Template, entry deploy.LogEntry, a *models.Application, d *models.Deployment, u *models.User) (string, error) {
	var summary bytes.Buffer

	var scheme string
	if config.SSLEnabled {
		scheme = "https"
	} else {
		scheme = "http"
	}

	var success bool
	if entry.EntryType == deploy.DEPLOYMENT_SUCCESS {
		success = true
	} else {
		success = false
	}

	deploymentUrl := fmt.Sprintf("%s://%s/%v/deployments/%v", scheme, config.Host,
		a.GitHubRepo, d.Id)

	gitHubUrl := fmt.Sprintf("https://github.com/%v/%v/commit/%v",
		a.GitHubOwner, a.GitHubRepo, d.CommitSha)

	err := t.Execute(&summary, map[string]interface{}{
		"GitHubRepo":    a.GitHubRepo,
		"Success":       success,
		"Branch":        d.Branch,
		"Target":        d.TargetName,
		"Username":      u.Name,
		"Comment":       d.Comment,
		"GitHubUrl":     gitHubUrl,
		"DeploymentURL": deploymentUrl,
	})

	return summary.String(), err
}
