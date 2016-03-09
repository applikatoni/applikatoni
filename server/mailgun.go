package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type MailgunClient struct {
	*http.Client
	requestURL string
	apiKey     string
}

func NewMailgunClient(baseURL, apiKey string) *MailgunClient {
	requestURL := fmt.Sprintf("%s/messages", baseURL)

	return &MailgunClient{
		Client:     &http.Client{},
		requestURL: requestURL,
		apiKey:     apiKey,
	}
}

func (m *MailgunClient) SendDigest(digest *DailyDigest) error {
	requestBody := m.newRequestBody(digest)
	request, err := m.newRequest(requestBody)
	if err != nil {
		return err
	}

	resp, err := m.Do(request)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("mailgun status code not 200. got=%d", resp.StatusCode)
	}

	return nil
}

func (m *MailgunClient) newRequestBody(digest *DailyDigest) io.Reader {
	params := url.Values{
		"from":    {fmt.Sprintf("%s <%s>", digestFromName, digestFromEmail)},
		"to":      {strings.Join(digest.Receivers, ",")},
		"subject": {digest.Subject},
		"text":    {digest.TextBody.String()},
		"html":    {digest.HtmlBody.String()},
	}

	return strings.NewReader(params.Encode())
}

func (m *MailgunClient) newRequest(body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest("POST", m.requestURL, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("api", m.apiKey)

	return req, nil
}
