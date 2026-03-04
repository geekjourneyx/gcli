package gmail

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"testing"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"

	"github.com/geekjourneyx/gcli/pkg/config"
	"github.com/geekjourneyx/gcli/pkg/errorsx"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestListMessagesAndGetMessage(t *testing.T) {
	var getCalls int
	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch {
			case req.Method == http.MethodGet && req.URL.Path == "/gmail/v1/users/me/messages":
				return jsonResponse(http.StatusOK, `{"messages":[{"id":"abc","threadId":"thread-1","snippet":"hello-list","internalDate":"1700000000000"}],"resultSizeEstimate":1}`), nil
			case req.Method == http.MethodGet && req.URL.Path == "/gmail/v1/users/me/messages/abc":
				getCalls++
				return jsonResponse(http.StatusOK, `{"id":"abc","threadId":"thread-1","snippet":"hello","internalDate":"1700000000000","payload":{"headers":[{"name":"From","value":"bot@example.com"},{"name":"Subject","value":"status"},{"name":"Date","value":"Tue, 01 Jan 2026 00:00:00 +0000"}]}}`), nil
			default:
				return jsonResponse(http.StatusNotFound, `{"error":{"code":404,"message":"Not Found"}}`), nil
			}
		}),
	}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test", TokenType: "Bearer"})
	client, err := NewClient(
		context.Background(),
		config.RuntimeConfig{APIEndpoint: "https://gmail.test.local"},
		tokenSource,
		option.WithHTTPClient(httpClient),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	page, err := client.ListMessages(context.Background(), ListOptions{Label: "INBOX", Limit: 1})
	if err != nil {
		t.Fatalf("ListMessages() error = %v", err)
	}
	if len(page.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(page.Messages))
	}
	if page.Messages[0].Subject != "" {
		t.Fatalf("unexpected subject without hydrate: %s", page.Messages[0].Subject)
	}
	if getCalls != 0 {
		t.Fatalf("expected zero users.messages.get calls, got %d", getCalls)
	}

	pageHydrated, err := client.ListMessages(context.Background(), ListOptions{Label: "INBOX", Limit: 1, Hydrate: true})
	if err != nil {
		t.Fatalf("ListMessages(Hydrate) error = %v", err)
	}
	if len(pageHydrated.Messages) != 1 {
		t.Fatalf("expected 1 hydrated message, got %d", len(pageHydrated.Messages))
	}
	if pageHydrated.Messages[0].Subject != "status" {
		t.Fatalf("unexpected hydrated subject: %s", pageHydrated.Messages[0].Subject)
	}
	if getCalls == 0 {
		t.Fatal("expected users.messages.get calls when hydrate=true")
	}

	msg, err := client.GetMessage(context.Background(), "abc", "metadata")
	if err != nil {
		t.Fatalf("GetMessage() error = %v", err)
	}
	if msg.ID != "abc" {
		t.Fatalf("unexpected id: %s", msg.ID)
	}
}

func TestGetMessageNotFound(t *testing.T) {
	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusNotFound, `{"error":{"code":404,"message":"Not Found"}}`), nil
		}),
	}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test", TokenType: "Bearer"})
	client, err := NewClient(
		context.Background(),
		config.RuntimeConfig{APIEndpoint: "https://gmail.test.local"},
		tokenSource,
		option.WithHTTPClient(httpClient),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.GetMessage(context.Background(), "missing", "metadata")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr := errorsx.From(err)
	if appErr.Code != errorsx.CodeMailNotFound {
		t.Fatalf("unexpected code: %s", appErr.Code)
	}
}

func TestGetMessageFullIncludesBody(t *testing.T) {
	textBody := base64.RawURLEncoding.EncodeToString([]byte("plain body content"))
	htmlBody := base64.RawURLEncoding.EncodeToString([]byte("<p>html body content</p>"))

	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method == http.MethodGet && req.URL.Path == "/gmail/v1/users/me/messages/fullmsg" && req.URL.Query().Get("format") == "full" {
				return jsonResponse(http.StatusOK, `{
					"id":"fullmsg",
					"threadId":"thread-full",
					"snippet":"hello",
					"internalDate":"1700000000000",
					"payload":{
						"headers":[
							{"name":"From","value":"alice@example.com"},
							{"name":"Subject","value":"body test"}
						],
						"mimeType":"multipart/alternative",
						"parts":[
							{"mimeType":"text/plain","body":{"data":"`+textBody+`"}},
							{"mimeType":"text/html","body":{"data":"`+htmlBody+`"}}
						]
					}
				}`), nil
			}
			return jsonResponse(http.StatusNotFound, `{"error":{"code":404,"message":"Not Found"}}`), nil
		}),
	}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test", TokenType: "Bearer"})
	client, err := NewClient(
		context.Background(),
		config.RuntimeConfig{APIEndpoint: "https://gmail.test.local"},
		tokenSource,
		option.WithHTTPClient(httpClient),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	msg, err := client.GetMessage(context.Background(), "fullmsg", "full")
	if err != nil {
		t.Fatalf("GetMessage() error = %v", err)
	}
	if msg.BodyText != "plain body content" {
		t.Fatalf("unexpected body text: %q", msg.BodyText)
	}
	if msg.BodyHTML != "<p>html body content</p>" {
		t.Fatalf("unexpected body html: %q", msg.BodyHTML)
	}
}

func TestGetMessageRawIncludesRawMIME(t *testing.T) {
	raw := "From: a@example.com\r\nTo: b@example.com\r\nSubject: raw\r\n\r\nhello raw"
	rawEncoded := base64.RawURLEncoding.EncodeToString([]byte(raw))

	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method == http.MethodGet && req.URL.Path == "/gmail/v1/users/me/messages/rawmsg" && req.URL.Query().Get("format") == "raw" {
				return jsonResponse(http.StatusOK, `{
					"id":"rawmsg",
					"threadId":"thread-raw",
					"snippet":"raw snippet",
					"internalDate":"1700000000000",
					"raw":"`+rawEncoded+`"
				}`), nil
			}
			return jsonResponse(http.StatusNotFound, `{"error":{"code":404,"message":"Not Found"}}`), nil
		}),
	}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test", TokenType: "Bearer"})
	client, err := NewClient(
		context.Background(),
		config.RuntimeConfig{APIEndpoint: "https://gmail.test.local"},
		tokenSource,
		option.WithHTTPClient(httpClient),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	msg, err := client.GetMessage(context.Background(), "rawmsg", "raw")
	if err != nil {
		t.Fatalf("GetMessage() error = %v", err)
	}
	if msg.RawMIME != raw {
		t.Fatalf("unexpected raw MIME: %q", msg.RawMIME)
	}
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
