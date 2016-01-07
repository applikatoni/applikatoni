package models

type Webhook struct {
	URL     string   `json:"url"`
	Entries []string `json:"entries"`
}

func (w *Webhook) IsEntryWanted(entry string) bool {
	for i := range w.Entries {
		if w.Entries[i] == entry {
			return true
		}
	}
	return false
}
