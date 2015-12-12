package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/applikatoni/applikatoni/models"
)

func commitLink(a *models.Application, sha string) string {
	return fmt.Sprintf("https://github.com/%s/%s/commit/%s",
		a.GitHubOwner, a.GitHubRepo, sha)
}

func fmtCommit(a *models.Application, d *models.Deployment) template.HTML {
	sha := d.CommitSha[:6]
	href := commitLink(a, d.CommitSha)

	if d.Branch == "" {
		return template.HTML("<a href=\"" + href + "\"><code>" + sha + "</code></a>")
	} else {
		return template.HTML("<a href=\"" + href + "\"><code>" + sha + " (" + d.Branch + ")</code></a>")
	}
}

func fmtDeploymentState(state models.DeploymentState) template.HTML {
	var s string

	switch state {
	case models.DEPLOYMENT_NEW:
		s = `<span data-attr="state-info" class="label label-primary">New</span>`
	case models.DEPLOYMENT_ACTIVE:
		s = `<span data-attr="state-info" class="label label-info">Active</span>`
	case models.DEPLOYMENT_SUCCESSFUL:
		s = `<span data-attr="state-info" class="label label-success">Successful</span>`
	case models.DEPLOYMENT_FAILED:
		s = `<span data-attr="state-info" class="label label-danger">Failed</span>`
	}

	return template.HTML(s)
}

func newlineToBreak(input string) template.HTML {
	output := template.HTMLEscapeString(input)
	return template.HTML(strings.Replace(output, "\n", "\n<br/>", -1))
}

func renderTemplate(w http.ResponseWriter, name string, data map[string]interface{}) {
	tmpl := templates[name]
	if tmpl == nil {
		log.Printf("template %s not found\n", name)
		return
	}

	data["Version"] = VERSION

	err := tmpl.Execute(w, data)
	if err != nil {
		log.Printf("rendering %s failed: %s\n", name, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func joinTemplatePaths(dir string, files []string) []string {
	joined := make([]string, len(files))
	for i := range files {
		joined[i] = filepath.Join(dir, files[i])
	}
	return joined
}

func parseTemplates(base string, templateSets [][]string) (map[string]*template.Template, error) {
	parsed := map[string]*template.Template{}

	for _, set := range templateSets {
		templateName := set[len(set)-1]
		t := template.New(templateName)

		t.Funcs(template.FuncMap{
			"fmtCommit":          fmtCommit,
			"fmtDeploymentState": fmtDeploymentState,
			"newlineToBreak":     newlineToBreak,
		})

		paths := joinTemplatePaths(base, set)

		_, err := t.ParseFiles(paths...)
		if err != nil {
			for _, p := range paths {
				data, err := Asset(p)
				if err != nil {
					return nil, err
				}
				_, err = t.Parse(string(data))
				if err != nil {
					return nil, err
				}
			}
		}

		t = t.Lookup("layout")
		if t == nil {
			return nil, fmt.Errorf("layout not found in %v", set)
		}
		parsed[templateName] = t
	}

	return parsed, nil
}
