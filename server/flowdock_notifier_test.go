package main

import (
	"net/http"
	"net/http/httptest"
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
