package main

import (
	"log"
	"net/http"
	"net/url"
	"text/template"

	"github.com/applikatoni/applikatoni/deploy"
	"github.com/applikatoni/applikatoni/models"

	"database/sql"
)

const flowdockTmplStr = `{{.GitHubRepo}} {{if .Success}}Successfully Deployed{{else}}Deploy Failed{{end}}:
**{{.Username}}** deployed **{{.Branch}}** on **{{.Target}}** :pizza:

> {{.Comment}}

[View latest commit on GitHub]({{.GitHubUrl}})
[Open deployment in Applikatoni]({{.DeploymentURL}})
`

var flowdockTemplate = template.Must(template.New("flowdockSummary").Parse(flowdockTmplStr))

func NotifyFlowdock(db *sql.DB, entry deploy.LogEntry) {
	deployment, err := getDeployment(db, entry.DeploymentId)
	if err != nil {
		log.Printf("Could not find deployment with id %v, %s\n", entry.DeploymentId, err)
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

	if target.FlowdockEndpoint == "" {
		return
	}

	user, err := getUser(db, deployment.UserId)
	if err != nil {
		log.Printf("Could not find User with id %v, %s\n", deployment.UserId, err)
		return
	}

	summary, err := generateSummary(flowdockTemplate, entry, application, deployment, user)
	if err != nil {
		log.Printf("Could not generate deployment summary, %s\n", err)
		return
	}

	SendFlowdockRequest(target.FlowdockEndpoint, deployment, summary)
}

func SendFlowdockRequest(endpoint string, d *models.Deployment, summary string) {
	params := url.Values{
		"event":   {"message"},
		"content": {summary},
		"tags":    {"deploy,applikatoni"},
	}

	resp, err := http.PostForm(endpoint, params)
	if err != nil {
		log.Printf("Notifying Flowdock failed (%s on %s, %s): err=%s\n",
			d.ApplicationName, d.TargetName, d.CommitSha, err)
		return
	}
	if resp.StatusCode != 201 {
		log.Printf("Notifying Flowdock failed (%s on %s, %s): status=%d\n",
			d.ApplicationName, d.TargetName, d.CommitSha, resp.StatusCode)
		return
	}

	log.Printf("Successfully notified Flowdock about deployment of %s on %s, %s!\n",
		d.ApplicationName, d.TargetName, d.CommitSha)
}

func newFlowdockNotifier(db *sql.DB) deploy.Listener {
	fn := func(logs <-chan deploy.LogEntry) {
		for entry := range logs {
			if entry.EntryType == deploy.DEPLOYMENT_SUCCESS || entry.EntryType == deploy.DEPLOYMENT_FAIL {
				go NotifyFlowdock(db, entry)
			}
		}
	}

	return fn
}
