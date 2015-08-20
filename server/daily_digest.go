package main

import (
	"bytes"
	"database/sql"
	"fmt"
	htmltemplate "html/template"
	"log"
	"path/filepath"
	"text/template"
	"time"

	"github.com/flinc/applikatoni/models"
)

const (
	digestSleepTime            = 1 * time.Minute
	digestHourOfDay            = 22
	digestInterval             = 24 * time.Hour
	mandrillMessagesEndpoint   = "https://mandrillapp.com/api/1.0/messages/send.json"
	digestTimezone             = "Europe/Berlin"
	digestSubjectFmt           = " 🍕 Applikatoni Daily Digest - %s"
	digestFromName             = "Applikatoni"
	digestFromEmail            = "no-reply@toni.flinc.org"
	digestHtmlTemplateDir      = "./assets/templates/"
	digestHtmlTemplateFilename = "daily_digest.tmpl"
	digestTextTemplate         = `Hello there!

Check out what the team behind {{.Application.Name}} deployed in the last 24 hours:

{{ range .Deployments }}
{{.CreatedAt.Format "02.01.2006 15:04 (MST)"}} -- {{.User.Name}} deployed to {{.TargetName}} with the following message:
    {{.Comment}}
{{ end}}

Always at your service:
your Applikatoni Daily Digest Team

Applikatoni - Deployments Al Forno
`
)

var (
	nextDailyDigest time.Time
)

type DailyDigest struct {
	FromName  string
	FromEmail string
	Receivers []string
	Subject   string
	TextBody  bytes.Buffer
	HtmlBody  bytes.Buffer
}

func SendDailyDigests(db *sql.DB) {
	nextDailyDigest = calcInitialDailyDigest(digestHourOfDay)

	for {
		now := time.Now()

		if now.After(nextDailyDigest) {
			log.Println("Sending daily digests...")

			for _, app := range config.Applications {
				err := sendApplicationDigest(db, app)
				if err != nil {
					log.Printf("Sending digest for application %s failed: %s", app.Name, err)
				}
			}

			nextDailyDigest = nextDailyDigest.Add(digestInterval)
		}

		time.Sleep(digestSleepTime)
	}
}

func sendApplicationDigest(db *sql.DB, a *models.Application) error {
	targetName := a.DailyDigestTarget
	receivers := a.DailyDigestReceivers
	since := time.Now().Add(-1 * digestInterval)

	if len(receivers) == 0 || targetName == "" {
		return nil
	}

	deployments, err := getDailyDigestDeployments(db, a, targetName, since)
	if err != nil {
		return err
	}

	if len(deployments) == 0 {
		log.Printf("Skipping daily digest for %s -- no deployments\n", a.Name)
		return nil
	}

	log.Printf("Sending daily digest for %s\n", a.Name)

	err = localizeTimestamps(deployments)
	if err != nil {
		return err
	}

	err = loadDeploymentsUsers(db, deployments)
	if err != nil {
		return err
	}

	digest, err := NewDigest(receivers, a, deployments)
	if err != nil {
		log.Printf("generating digest for %s failed: %s\n", a.Name, err)
		return err
	}

	err = SendDigestMail(digest)
	if err != nil {
		log.Printf("sending digest for %s to %s failed: %s\n", a.Name, receivers, err)
		return err
	}

	log.Printf("successfully sent daily digest for %s to %s\n", a.Name, receivers)
	return nil
}

func NewDigest(receivers []string, a *models.Application, deployments []*models.Deployment) (*DailyDigest, error) {
	textBody, err := generateDigestTextBody(a, deployments)
	if err != nil {
		return nil, err
	}

	htmlBody, err := generateDigestHtmlBody(a, deployments)
	if err != nil {
		return nil, err
	}

	digest := &DailyDigest{
		FromName:  digestFromName,
		FromEmail: digestFromEmail,
		Receivers: receivers,
		TextBody:  textBody,
		HtmlBody:  htmlBody,
		Subject:   fmt.Sprintf(digestSubjectFmt, a.Name),
	}
	return digest, nil
}

func calcInitialDailyDigest(hourOfDay int) time.Time {
	year, month, day := time.Now().Date()
	return time.Date(year, month, day, hourOfDay, 0, 0, 0, time.Local)
}

func localizeTimestamps(deployments []*models.Deployment) error {
	timezone, err := time.LoadLocation(digestTimezone)
	if err != nil {
		return err
	}

	for _, d := range deployments {
		d.CreatedAt = d.CreatedAt.In(timezone)
	}

	return nil
}

func generateDigestTextBody(a *models.Application, deployments []*models.Deployment) (bytes.Buffer, error) {
	var digestTextBody bytes.Buffer

	tmpl, err := template.New("digestTextBody").Parse(digestTextTemplate)
	if err != nil {
		return digestTextBody, err
	}

	vars := map[string]interface{}{
		"Application": a,
		"Deployments": deployments,
	}

	if err = tmpl.Execute(&digestTextBody, vars); err != nil {
		return digestTextBody, err
	}

	return digestTextBody, nil
}

func generateDigestHtmlBody(a *models.Application, deployments []*models.Deployment) (bytes.Buffer, error) {
	var digestHtmlBody bytes.Buffer
	path := filepath.Join(digestHtmlTemplateDir, digestHtmlTemplateFilename)

	tmpl := htmltemplate.New("")
	tmpl.Funcs(htmltemplate.FuncMap{"newlineToBreak": newlineToBreak})

	tmpl, err := tmpl.ParseFiles(path)
	if err != nil {
		return digestHtmlBody, err
	}

	vars := map[string]interface{}{
		"Application": a,
		"Deployments": deployments,
	}

	err = tmpl.ExecuteTemplate(&digestHtmlBody, digestHtmlTemplateFilename, vars)
	if err != nil {
		return digestHtmlBody, err
	}

	return digestHtmlBody, nil
}
