package models

type Host struct {
	Name  string   `json:"name"`
	Roles []string `json:"roles"`
}
