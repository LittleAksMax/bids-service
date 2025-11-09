package main

import (
	"encoding/json"
	"net/http"
	"os"
)

type RefreshToken struct {
	refreshToken string
}

func (rft *RefreshToken) Get() string {
	if rft.refreshToken != "" {
		return rft.refreshToken
	}

	client := &http.Client{}
	uri := os.Getenv("REFRESH_TOKEN_URI")
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return ""
	}

	req.Header.Set("X-Api-Key", os.Getenv("REFRESH_TOKEN_URI_ACCESS_KEY"))
	res, err := client.Do(req)
	if err != nil {
		return ""
	}
	if res.StatusCode != http.StatusOK {
		return ""
	}
	defer res.Body.Close()

	target := struct {
		RefreshToken string `json:"refresh_token"`
	}{}

	err = json.NewDecoder(res.Body).Decode(&target)
	if err != nil {
		return ""
	}

	rft.refreshToken = target.RefreshToken
	return rft.refreshToken
}
