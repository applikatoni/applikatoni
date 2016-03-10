package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMandrillSendDigest(t *testing.T) {
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

	writeFakeResponse := func(w http.ResponseWriter) {
		fakeResponse := []MandrillDeliveryStatus{
			{Email: digest.Receivers[0], Status: "sent"},
		}
		js, err := json.Marshal(fakeResponse)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPayload := &MandrillMessagePayload{}

		err := json.NewDecoder(r.Body).Decode(receivedPayload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if receivedPayload.Key != "mandrillapikey" {
			t.Errorf("received wrong api key. got=%s", receivedPayload.Key)
			w.WriteHeader(500)
			return
		}
		if receivedPayload.Message.To[0].Email != digest.Receivers[0] {
			t.Errorf("wrong receivers. got=%+v", receivedPayload.Message.To)
			w.WriteHeader(500)
			return
		}
		if receivedPayload.Message.FromEmail != digestFromEmail {
			t.Errorf("FromMail wrong. got=%s", receivedPayload.Message.FromEmail)
			w.WriteHeader(500)
			return
		}
		if receivedPayload.Message.FromName != digestFromName {
			t.Errorf("FromName wrong. got=%s", receivedPayload.Message.FromName)
			w.WriteHeader(500)
			return
		}
		if receivedPayload.Message.Subject != digest.Subject {
			t.Errorf("Subject wrong. got=%s", receivedPayload.Message.Subject)
			w.WriteHeader(500)
			return
		}
		if receivedPayload.Message.Text != digest.TextBody.String() {
			t.Errorf("Text wrong. got=%s", receivedPayload.Message.Text)
			w.WriteHeader(500)
			return
		}
		if receivedPayload.Message.Html != digest.HtmlBody.String() {
			t.Errorf("Html wrong. got=%s", receivedPayload.Message.Html)
			w.WriteHeader(500)
			return
		}

		writeFakeResponse(w)
	}))
	defer ts.Close()

	mandrill := NewMandrillClient(ts.URL, "mandrillapikey")
	err := mandrill.SendDigest(digest)
	if err != nil {
		t.Errorf("SendDigest error: %s", err)
	}
}
