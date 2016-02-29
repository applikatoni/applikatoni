package main

import (
	"bytes"
	"database/sql"
	"log"
	"net/http"
	"net/url"
	"text/template"

	"github.com/applikatoni/applikatoni/deploy"
	"github.com/applikatoni/applikatoni/models"
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
		SendNewRelicRequest(newRelicNotifyEndpoint, ev.Entry, ev.Deployment, ev.Target, ev.Application, ev.User)
	}
}

func SendNewRelicRequest(endpoint string, e deploy.LogEntry, d *models.Deployment, t *models.Target, a *models.Application, u *models.User) {
	summary, err := generateSummary(newRelicTemplate, e, a, d, u)
	if err != nil {
		log.Printf("Could not generate deployment summary, %s\n", err)
		return
	}

	data := url.Values{}
	data.Set("deployment[app_id]", t.NewRelicAppId)
	data.Set("deployment[description]", d.Comment)
	data.Set("deployment[revision]", d.CommitSha)
	data.Set("deployment[user]", u.Name)
	data.Set("deployment[changelog]", summary)

	client := &http.Client{}

	// post URL-encoded payload, must satisfy io interface
	req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
	req.Header.Set("x-api-key", t.NewRelicApiKey)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Notifying NewRelic failed (%s on %s, %s): err=%s\n",
			d.ApplicationName, d.TargetName, d.CommitSha, err)
		return
	}
	if resp.StatusCode != 201 {
		log.Printf("Notifying NewRelic failed (%s on %s, %s): status=%d\n",
			d.ApplicationName, d.TargetName, d.CommitSha, resp.StatusCode)
		return
	}

	log.Printf("Successfully notified New Relic about deployment of %v on %v, %v!\n", d.ApplicationName, d.TargetName, d.CommitSha)
}
