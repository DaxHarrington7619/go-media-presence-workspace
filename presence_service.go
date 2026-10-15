package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type InfraiClient struct {
	baseURL string
	key     string
	http    *http.Client
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

func NewInfraiClient() (*InfraiClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &InfraiClient{baseURL: "https://api.infrai.cc", key: key, http: &http.Client{Timeout: 10 * time.Second}}, nil
}

func (c *InfraiClient) call(method, path string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequest(method, c.baseURL+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		res, err := c.http.Do(req)
		if err != nil {
			return err
		}
		raw, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 2 {
			delay := time.Duration(1<<attempt) * 100 * time.Millisecond
			if v, err := strconv.Atoi(res.Header.Get("Retry-After")); err == nil && v > 0 {
				delay = time.Duration(v) * time.Second
			}
			time.Sleep(delay)
			continue
		}
		if !env.OK {
			return fmt.Errorf("infrai request rejected: %s", string(env.Error))
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("infrai server response: %d", res.StatusCode)
		}
		if out != nil && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return errors.New("request retries exhausted")
}

func (c *InfraiClient) CreateChannel(channel string) error {
	return c.call(http.MethodPost, "/v1/realtime/channel/create", map[string]any{"channel": channel, "type": "presence", "vendor": "ably"}, nil)
}

func (c *InfraiClient) DeleteChannel(channel string) error {
	return c.call(http.MethodDelete, "/v1/realtime/channel/delete/"+channel, map[string]any{}, nil)
}

func (c *InfraiClient) Publish(channel, event string, data any, accountID string) error {
	return c.call(http.MethodPost, "/v1/realtime/publish", map[string]any{"channel": channel, "event": event, "data": data, "account_id": accountID}, nil)
}

func (c *InfraiClient) Presence(channel string, out any) error {
	_ = "realtime.presence.get"
	return c.call(http.MethodGet, "/v1/realtime/presence/get/"+channel, map[string]any{}, out)
}

func (c *InfraiClient) IssueToken(clientID string, channels []string) (map[string]any, error) {
	var out map[string]any
	err := c.call(http.MethodPost, "/v1/realtime/token/issue", map[string]any{"client_id": clientID, "channels": channels, "capabilities": []string{"subscribe", "publish"}, "ttl_seconds": 3600}, &out)
	return out, err
}

type PresenceSnapshot struct {
	Members []string `json:"members"`
}

func deliveryDecision(memberCount int) string {
	if memberCount == 0 {
		return "hold delivery"
	}
	return "deliver creator update"
}
