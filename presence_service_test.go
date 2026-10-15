package main

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestIngestDeletesChannel(t *testing.T) {
	for _, failPublish := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "publish failure"}[failPublish], func(t *testing.T) {
			var paths []string
			client := &InfraiClient{baseURL: "https://example.test", key: "test"}
			client.http = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				paths = append(paths, r.URL.Path)
				body := `{"ok":true,"data":{"members":[]}}`
				if failPublish && r.URL.Path == "/v1/realtime/publish" {
					body = `{"ok":false,"error":"publish failed"}`
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
			})}
			_, err := ingest(client, ingestRequest{Channel: "temporary"})
			if (err != nil) != failPublish {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := paths[len(paths)-1]; got != "/v1/realtime/channel/delete/temporary" {
				t.Fatalf("last call = %q, want channel deletion", got)
			}
		})
	}
}

func TestDeliveryDecision(t *testing.T) {
	cases := []struct {
		name    string
		members int
		want    string
	}{
		{"no creator online", 0, "hold delivery"},
		{"creator online", 1, "deliver creator update"},
		{"several creators", 4, "deliver creator update"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := deliveryDecision(tc.members); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
