package downloader

import (
	"net/http"
	"sync"
	"time"
)

var httpClient *http.Client
var once = &sync.Once{}

func GetHttpClient(timeout time.Duration) *http.Client {
	once.Do(func() {
		httpClient = &http.Client{
			Timeout: 0,
		}
	})
	return httpClient
}
