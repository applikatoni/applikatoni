package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type MailgunClient struct {
	*http.Client
	baseUrl string
	apiKey  string
}

func NewMailgunClient(baseUrl, apiKey string) *MailgunClient {
	return &MailgunClient{
		Client:  &http.Client{},
		baseUrl: baseUrl,
		apiKey:  apiKey,
	}
}

func (m *MailgunClient) SendDigest(digest *DailyDigest) error {
	params := url.Values{
		"from":    {fmt.Sprintf("%s <%s>", digestFromName, digestFromEmail)},
		"to":      {strings.Join(digest.Receivers, ",")},
		"subject": {digest.Subject},
		"text":    {digest.TextBody.String()},
		"html":    {digest.HtmlBody.String()},
	}

	requestBody := strings.NewReader(params.Encode())
	requestUrl := fmt.Sprintf("%s/messages", m.baseUrl)

	req, err := http.NewRequest("POST", requestUrl, requestBody)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("api", m.apiKey)

	resp, err := m.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("mailgun status code not 200. got=%d", resp.StatusCode)
	}

	return nil
}
