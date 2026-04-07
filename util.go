package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func httpClient(timeoutSeconds int) *http.Client {
	return &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second}
}

func checkStatus(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("HTTP %s: %s", resp.Status, body)
}
