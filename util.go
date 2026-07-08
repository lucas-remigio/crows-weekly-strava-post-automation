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

func fetchHTML(urlStr string, timeoutSeconds int) (string, error) {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "pt-PT,pt;q=0.9,en-US;q=0.8,en;q=0.7")

	client := httpClient(timeoutSeconds)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(bodyBytes), nil
}
