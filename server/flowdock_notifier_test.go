package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/applikatoni/applikatoni/models"
)

func TestSendFlowdockRequest(t *testing.T) {
	deployment := &models.Deployment{
		ApplicationName: "web",
		TargetName:      "staging",
		Branch:          "master",
		CommitSha:       "f00b4r",
	}
	summary := "Deployment was okay, I guess"

	tests := []struct {
		formKey  string
		expected string
	}{
		{"event", "message"},
		{"content", summary},
		{"tags", "deploy,applikatoni"},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, tt := range tests {
			actual := r.FormValue(tt.formKey)
			if actual != tt.expected {
				t.Errorf("sent wrong value for %s. want=%s, got=%s", tt.formKey, tt.expected, actual)
				w.WriteHeader(422)
				return
			}
		}
		w.WriteHeader(201)
	}))
	defer ts.Close()

	SendFlowdockRequest(ts.URL, deployment, summary)
}

func TestFlowdockSummary(t *testing.T) {
	var summary bytes.Buffer

	comment := `* First fix
* Second fix
* Third fix`

	err := flowdockTemplate.Execute(&summary, map[string]interface{}{
		"GitHubRepo":    "web-app",
		"Success":       true,
		"Branch":        "master",
		"Target":        "production",
		"Username":      "mrnugget",
		"Comment":       comment,
		"CommentLines":  strings.Split(comment, "\n"),
		"GitHubUrl":     "https://github.com/shipping-co/web-app",
		"DeploymentURL": "http://localhost:8080/web-app/deployments/1",
	})
	if err != nil {
		t.Errorf("Executing template failed: %s", err)
	}

	summaryStr := summary.String()

	if !strings.Contains(summaryStr, "> * First fix\n") {
		t.Errorf("Comment line not indented. got=%q", summaryStr)
	}

	if !strings.Contains(summaryStr, "> * Second fix\n") {
		t.Errorf("Comment line not indented. got=%q", summaryStr)
	}

	if !strings.Contains(summaryStr, "> * Third fix") {
		t.Errorf("Comment line not indented. got=%q", summaryStr)
	}
}
