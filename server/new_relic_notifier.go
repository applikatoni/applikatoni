package main

import (
	"bytes"
	"database/sql"
	"log"
	"net/http"
	"net/url"
	"text/template"
)

const (
	newRelicNotifyEndpoint = "https://api.newrelic.com/deployments.xml"
)

const newRelicTmplStr = `Deployed {{.GitHubRepo}}/{{.Branch}} on {{.Target}} by {{.Username}} :pizza:
{{.Comment}}
SHA: {{.GitHubUrl}}
URL: {{.DeploymentURL}}
`

var newRelicTemplate = template.Must(template.New("newRelicSummary").Parse(newRelicTmplStr))

func NotifyNewRelic(db *sql.DB, ev *DeploymentEvent) {
	if ev.Target.NewRelicApiKey != "" && ev.Target.NewRelicAppId != "" {
		SendNewRelicRequest(newRelicNotifyEndpoint, ev)
	}
}

func SendNewRelicRequest(endpoint string, ev *DeploymentEvent) {
	summary, err := generateSummary(newRelicTemplate, ev)
	if err != nil {
		log.Printf("Could not generate deployment summary, %s\n", err)
		return
	}

	data := url.Values{}
	data.Set("deployment[app_id]", ev.Target.NewRelicAppId)
	data.Set("deployment[description]", ev.Deployment.Comment)
	data.Set("deployment[revision]", ev.Deployment.CommitSha)
	data.Set("deployment[user]", ev.User.Name)
	data.Set("deployment[changelog]", summary)

	client := &http.Client{}

	// post URL-encoded payload, must satisfy io interface
	req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
	req.Header.Set("x-api-key", ev.Target.NewRelicApiKey)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Notifying NewRelic failed (%s on %s, %s): err=%s\n",
			ev.Application.Name, ev.Target.Name, ev.Deployment.CommitSha, err)
		return
	}
	if resp.StatusCode != 201 {
		log.Printf("Notifying NewRelic failed (%s on %s, %s): status=%d\n",
			ev.Application.Name, ev.Target.Name, ev.Deployment.CommitSha,
			resp.StatusCode)
		return
	}

	log.Printf("Successfully notified New Relic about deployment of %v on %v, %v!\n",
		ev.Application.Name, ev.Target.Name, ev.Deployment.CommitSha)
}
