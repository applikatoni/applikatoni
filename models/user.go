package models

type User struct {
	Name        string `json:"login"`
	Id          int    `json:"id"`
	AccessToken string
	AvatarUrl   string `json:"avatar_url"`
	ApiToken    string
}
