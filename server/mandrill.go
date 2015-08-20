package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

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

func SendDigestMail(digest *DailyDigest) error {
	to := []MandrillReceiver{}
	for _, email := range digest.Receivers {
		to = append(to, MandrillReceiver{email})
	}

	payload := &MandrillMessagePayload{}
	payload.Key = config.MandrillAPIKey
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
		return err
	}

	client := &http.Client{}

	req, err := http.NewRequest("POST", mandrillMessagesEndpoint, bytes.NewReader(data))
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("mandrill response code not 200, but %d", resp.StatusCode)
	}

	responseBody := []MandrillDeliveryStatus{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	if err != nil {
		return err
	}

	for _, delivery := range responseBody {
		if delivery.Status != "sent" {
			log.Printf("sending email to %s failed. delivery status: %s", delivery.Email, delivery.Status)
		}
	}

	return nil
}
