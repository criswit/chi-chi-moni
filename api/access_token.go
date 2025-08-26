package api

import (
	"encoding/base64"
	"io"
	"net/http"
)

type AccessToken struct {
	Username string
	Password string
	Url      string
}
type AccessTokenResolver struct {
	setupToken string
}

func (r *AccessTokenResolver) Resolve() (AccessToken, error) {
	decoded, err := base64.StdEncoding.DecodeString(r.setupToken)
	if err != nil {
		return AccessToken{}, err
	}
	claimUrl := string(decoded)
	resp, err := http.Post(claimUrl, "application/json", nil)
	if err != nil {
		return AccessToken{}, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return AccessToken{}, err
	}
	accessUrl := string(b)
	// Example: https://username:password@host/path
	// Strip "https://"
	const prefix = "https://"
	if len(accessUrl) < len(prefix) || accessUrl[:len(prefix)] != prefix {
		return AccessToken{}, err
	}
	rest := accessUrl[len(prefix):]
	// Find the first '@'
	atIdx := -1
	for i, c := range rest {
		if c == '@' {
			atIdx = i
			break
		}
	}
	if atIdx == -1 {
		return AccessToken{}, err
	}
	auth := rest[:atIdx]
	url := rest[atIdx+1:]
	// Split auth into username and password
	colonIdx := -1
	for i, c := range auth {
		if c == ':' {
			colonIdx = i
			break
		}
	}
	if colonIdx == -1 {
		return AccessToken{}, err
	}
	username := auth[:colonIdx]
	password := auth[colonIdx+1:]
	return AccessToken{
		Username: username,
		Password: password,
		Url:      url,
	}, nil

}

func NewAccessTokenResolver(setupToken string) *AccessTokenResolver {
	return &AccessTokenResolver{setupToken: setupToken}
}
