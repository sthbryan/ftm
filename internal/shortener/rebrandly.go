package shortener

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const rebrandlyAPI = "https://api.rebrandly.com/v1/links"

type RebrandlyClient struct {
	apiKey string
	client *http.Client
}

type rebrandlyRequest struct {
	Destination string `json:"destination"`
	Slashtag    string `json:"slashtag,omitempty"`
	Domain      struct {
		FullName string `json:"fullName"`
	} `json:"domain"`
}

type rebrandlyResponse struct {
	ID          string `json:"id"`
	ShortURL    string `json:"shortUrl"`
	Destination string `json:"destination"`
}

func NewRebrandly(apiKey string) *RebrandlyClient {
	return &RebrandlyClient{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *RebrandlyClient) Name() string {
	return "rebrandly"
}

func (c *RebrandlyClient) Shorten(longURL, custom string) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("rebrandly requires API key")
	}

	reqBody := rebrandlyRequest{
		Destination: longURL,
		Slashtag:    custom,
	}
	reqBody.Domain.FullName = "rebrand.ly"

	jsonBody, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", rebrandlyAPI, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to contact rebrandly: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return "", ShortenError{
			Reason:  "API_ERROR",
			Message: string(body),
		}
	}

	var result rebrandlyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return "https://" + result.ShortURL, nil
}
