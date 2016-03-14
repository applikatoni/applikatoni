package main

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/applikatoni/applikatoni/models"
)

func generateSummary(t *template.Template, ev *DeploymentEvent) (string, error) {
	var success bool
	if ev.State == models.DEPLOYMENT_SUCCESSFUL {
		success = true
	} else {
		success = false
	}

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
		"CommentLines":  strings.Split(ev.Deployment.Comment, "\n"),
		"GitHubUrl":     gitHubUrl,
		"DeploymentURL": ev.DeploymentURL(),
	})

	return summary.String(), err
}
