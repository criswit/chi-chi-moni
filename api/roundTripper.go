package api

import "net/http"

type SimpleFinRoundTripper struct {
	username string
	password string
	Base     http.RoundTripper
}

func (rt *SimpleFinRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.SetBasicAuth(rt.username, rt.password)
	return rt.base().RoundTrip(cloned)
}

func (rt *SimpleFinRoundTripper) base() http.RoundTripper {
	if rt.Base != nil {
		return rt.Base
	}
	return http.DefaultTransport
}
