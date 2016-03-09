package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const mandrillMessagesEndpoint = "https://mandrillapp.com/api/1.0/messages/send.json"

type MandrillMessage struct {
	Html      string             `json:"html"`
	Text      string             `json:"text"`
	Subject   string             `json:"subject"`
	FromEmail string             `json:"from_email"`
	FromName  string             `json:"from_name"`
	To        []MandrillReceiver `json:"to"`
}
type MandrillReceiver struct {
	Email string `json:"email"`
}

type MandrillMessagePayload struct {
	Key     string          `json:"key"`
	Message MandrillMessage `json:"message"`
}

type MandrillDeliveryStatus struct {
	Email  string `json:"email"`
	Status string `json:"status"`
}

type MandrillClient struct {
	*http.Client
	endpoint string
	apiKey   string
}

func NewMandrillClient(endpoint, apiKey string) *MandrillClient {
	return &MandrillClient{
		Client:   &http.Client{},
		endpoint: endpoint,
		apiKey:   apiKey,
	}
}

func (m *MandrillClient) SendDigest(digest *DailyDigest) error {
	requestBody, err := m.newRequestBody(digest)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", m.endpoint, requestBody)
	if err != nil {
		return err
	}

	resp, err := m.Do(req)
	if err != nil {
		return err
	}

	return m.checkResponseStatus(resp)
}

func (m *MandrillClient) newRequestBody(digest *DailyDigest) (io.Reader, error) {
	to := []MandrillReceiver{}
	for _, email := range digest.Receivers {
		to = append(to, MandrillReceiver{email})
	}

	payload := &MandrillMessagePayload{}
	payload.Key = m.apiKey
	payload.Message = MandrillMessage{
		Html:      digest.HtmlBody.String(),
		Text:      digest.TextBody.String(),
		Subject:   digest.Subject,
		FromEmail: digest.FromEmail,
		FromName:  digest.FromName,
		To:        to,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(data), nil
}

func (m *MandrillClient) checkResponseStatus(response *http.Response) error {
	if response.StatusCode != 200 {
		return fmt.Errorf("mandrill response code is %d", response.StatusCode)
	}

	responseBody := []MandrillDeliveryStatus{}
	err := json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return err
	}

	for _, delivery := range responseBody {
		if delivery.Status != "sent" {
			err := fmt.Errorf("Sending digest to %s failed. delivery status=%s",
				delivery.Email, delivery.Status)
			return err
		}
	}

	return nil
}
