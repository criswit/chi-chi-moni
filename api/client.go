package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/criswit/chi-chi-moni/model"
)

type SimpleFinClient struct {
	client  *http.Client
	baseUrl string
}

type GetAccountsOptions struct {
	StartDate    *int64   // Unix epoch timestamp - transactions on or after this date
	EndDate      *int64   // Unix epoch timestamp - transactions before (but not on) this date
	Pending      bool     // Include pending transactions (default: false)
	AccountIDs   []string // Filter to specific account IDs
	BalancesOnly bool     // Return only balances, no transaction data
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

func (c *SimpleFinClient) GetAccounts(opts *GetAccountsOptions) (*model.GetAccountsResponse, error) {
	params := url.Values{}
	
	if opts != nil {
		if opts.StartDate != nil {
			params.Add("start-date", strconv.FormatInt(*opts.StartDate, 10))
		}
		if opts.EndDate != nil {
			params.Add("end-date", strconv.FormatInt(*opts.EndDate, 10))
		}
		if opts.Pending {
			params.Add("pending", "1")
		}
		for _, accountID := range opts.AccountIDs {
			params.Add("account", accountID)
		}
		if opts.BalancesOnly {
			params.Add("balances-only", "1")
		} else {
			params.Add("balances-only", "0")
		}
	} else {
		params.Add("balances-only", "0")
	}
	
	queryString := params.Encode()
	accountsURL := fmt.Sprintf("https://%s/accounts", c.baseUrl)
	if queryString != "" {
		accountsURL = fmt.Sprintf("%s?%s", accountsURL, queryString)
	}
	
	resp, err := c.client.Get(accountsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var response model.GetAccountsResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, err
	}
	return &response, nil
}
