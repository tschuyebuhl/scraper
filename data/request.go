package data

import "net/http"

type HTTPRequester interface {
	Get(string) (*http.Response, error)
}
