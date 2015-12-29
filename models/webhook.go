package models

import "regexp"

type Webhook struct {
	URL         string   `json:"url"`
	EntryFilter []string `json:"filter"`
}

func (w *Webhook) IsEntryWanted(entry string) bool {
	for _, filter := range w.EntryFilter {
		if matched, _ := regexp.MatchString(filter, entry); matched {
			return matched
		}
	}
	return false
}
