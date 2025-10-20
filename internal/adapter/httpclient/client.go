package httpclient

import (
	"net/http"
	"time"
)

func NewHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 15 * time.Second,
	}
}
