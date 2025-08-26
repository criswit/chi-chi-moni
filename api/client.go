package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/criswit/chi-chi-moni/model"
)

type SimpleFinClient struct {
	client  *http.Client
	baseUrl string
}

func NewSimpleFinClient(accessToken AccessToken) (*SimpleFinClient, error) {
	rt := &SimpleFinRoundTripper{
		username: accessToken.Username,
		password: accessToken.Password,
	}
	return &SimpleFinClient{
		client: &http.Client{
			Transport: rt,
		},
		baseUrl: accessToken.Url,
	}, nil
}

func (c *SimpleFinClient) GetAccounts() (*model.GetAccountsResponse, error) {
	url := fmt.Sprintf("https://%s/accounts", c.baseUrl)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var financialResponse *model.GetAccountsResponse
	if err := json.Unmarshal(b, &financialResponse); err != nil {
		return nil, err
	}
	return financialResponse, nil
}
