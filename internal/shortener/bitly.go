package shortener

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const bitlyAPI = "https://api-ssl.bitly.com/v4/shorten"
const bitlyUpdateAPI = "https://api-ssl.bitly.com/v4/bitlinks/"

type BitlyClient struct {
	apiKey string
	client *http.Client
}

type bitlyCreateRequest struct {
	LongURL string `json:"long_url"`
	Domain  string `json:"domain,omitempty"`
}

type bitlyUpdateRequest struct {
	LongURL string `json:"long_url"`
}

type bitlyResponse struct {
	ID      string `json:"id"`
	Link    string `json:"link"`
	LongURL string `json:"long_url"`
}

func NewBitly(apiKey string) *BitlyClient {
	return &BitlyClient{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *BitlyClient) Name() string {
	return "bitly"
}

func (c *BitlyClient) Shorten(longURL, custom string) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("bitly requires API key. Get one at https://app.bitly.com/settings/api/")
	}

	reqBody := bitlyCreateRequest{
		LongURL: longURL,
	}

	jsonBody, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", bitlyAPI, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to contact bitly: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == 409 {
		return "", ShortenError{
			Reason:  "ALREADY_EXISTS",
			Message: "URL already shortened, use Update instead",
		}
	}

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return "", ShortenError{
			Reason:  "API_ERROR",
			Message: string(body),
		}
	}

	var result bitlyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Link, nil
}

func (c *BitlyClient) Update(shortURL, newLongURL string) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("bitly requires API key")
	}

	id := extractBitlinkID(shortURL)
	if id == "" {
		return "", fmt.Errorf("invalid bitly URL")
	}

	reqBody := bitlyUpdateRequest{
		LongURL: newLongURL,
	}

	jsonBody, _ := json.Marshal(reqBody)

	url := bitlyUpdateAPI + id
	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to update bitly: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", ShortenError{
			Reason:  "UPDATE_FAILED",
			Message: string(body),
		}
	}

	var result bitlyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Link, nil
}

func extractBitlinkID(shortURL string) string {
	if idx := len("https://"); len(shortURL) > idx && shortURL[:idx] == "https://" {
		return shortURL[idx:]
	}
	if idx := len("http://"); len(shortURL) > idx && shortURL[:idx] == "http://" {
		return shortURL[idx:]
	}
	return shortURL
}

func IsBitlyUpdateable(err error) bool {
	if se, ok := err.(ShortenError); ok {
		return se.Reason == "ALREADY_EXISTS"
	}
	return false
}
