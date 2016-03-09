package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMailgunSendDigest(t *testing.T) {
	var digestHtmlBody bytes.Buffer
	digestHtmlBody.WriteString("<h1>Hello there!")

	var digestTextBody bytes.Buffer
	digestTextBody.WriteString("Hello there!")

	digest := &DailyDigest{
		FromName:  digestFromName,
		FromEmail: digestFromEmail,
		Subject:   "foobar",
		Receivers: []string{"mrnugget@gmail.com"},
		TextBody:  digestTextBody,
		HtmlBody:  digestHtmlBody,
	}

	tests := []struct {
		formKey  string
		expected string
	}{
		{"from", fmt.Sprintf("%s <%s>", digestFromName, digestFromEmail)},
		{"to", strings.Join(digest.Receivers, ",")},
		{"subject", digest.Subject},
		{"text", digestTextBody.String()},
		{"html", digestHtmlBody.String()},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, tt := range tests {
			actual := r.FormValue(tt.formKey)
			if actual != tt.expected {
				t.Errorf("sent wrong value for %s. want=%s, got=%s", tt.formKey, tt.expected, actual)
			}
		}
		w.WriteHeader(200)
	}))
	defer ts.Close()

	mailgun := NewMailgunClient(ts.URL, "foobarapikey")
	err := mailgun.SendDigest(digest)
	if err != nil {
		t.Errorf("SendDigest error: %s", err)
	}
}
