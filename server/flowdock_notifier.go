package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"

	"github.com/applikatoni/applikatoni/deploy"
	"github.com/applikatoni/applikatoni/models"

	"database/sql"
)

const summaryTmplStr = `Deployed {{.GitHubRepo}}/{{.Branch}} on {{.Target}} by {{.Username}} :pizza:

{{.Comment}}

SHA: {{.GitHubUrl}}
URL: {{.DeploymentURL}}
`

var summaryTemplate = template.Must(template.New("summary").Parse(summaryTmplStr))

func NotifyFlowdock(deploymentId int) {
	deployment, err := getDeployment(db, deploymentId)
	if err != nil {
		log.Printf("Could not find deployment with id %v, %s\n", deploymentId, err)
		return
	}

	application, err := findApplication(deployment.ApplicationName)
	if err != nil {
		log.Printf("Could not find application with name %v, %s\n", deployment.ApplicationName, err)
		return
	}

	target, err := findTarget(application, deployment.TargetName)
	if err != nil {
		log.Printf("Could not find target with name %v, %s\n", deployment.TargetName, err)
		return
	}

	user, err := getUser(db, deployment.UserId)
	if err != nil {
		log.Printf("Could not find User with id %v, %s\n", deployment.UserId, err)
		return
	}

	if target.FlowdockEndpoint == "" {
		return
	}

	summary, err := generateSummary(application, deployment, user)
	if err != nil {
		log.Printf("Could not generate deployment summary, %s\n", err)
	}

	params := url.Values{
		"event":   {"message"},
		"content": {summary},
		"tags":    {"deploy,applikatoni"},
	}

	resp, err := http.PostForm(target.FlowdockEndpoint, params)

	if err != nil || resp.StatusCode != 201 {
		log.Printf("Error while notifying Flowdock about deployment of %v on %v, %v! err: %s, resp: %s\n", deployment.ApplicationName,
			deployment.TargetName,
			deployment.CommitSha,
			err,
			resp.Status)
	} else {
		log.Printf("Successfully notified Flowdock about deployment of %v on %v, %v!\n", deployment.ApplicationName, deployment.TargetName, deployment.CommitSha)
	}
}

func generateSummary(a *models.Application, d *models.Deployment, u *models.User) (string, error) {
	var summary bytes.Buffer

	var scheme string
	if config.SSLEnabled {
		scheme = "https"
	} else {
		scheme = "http"
	}

	deploymentUrl := fmt.Sprintf("%s://%s/%v/deployments/%v", scheme, config.Host,
		a.GitHubRepo, d.Id)

	gitHubUrl := fmt.Sprintf("https://github.com/%v/%v/commit/%v",
		a.GitHubOwner, a.GitHubRepo, d.CommitSha)

	err := summaryTemplate.Execute(&summary, map[string]interface{}{
		"GitHubRepo":    a.GitHubRepo,
		"Branch":        d.Branch,
		"Target":        d.TargetName,
		"Username":      u.Name,
		"Comment":       d.Comment,
		"GitHubUrl":     gitHubUrl,
		"DeploymentURL": deploymentUrl,
	})

	return summary.String(), err
}

func newFlowdockNotifier(db *sql.DB) deploy.Listener {
	fn := func(logs <-chan deploy.LogEntry) {
		for entry := range logs {
			if entry.EntryType == deploy.DEPLOYMENT_SUCCESS {
				go NotifyFlowdock(entry.DeploymentId)
			}
		}
	}

	return fn
}
