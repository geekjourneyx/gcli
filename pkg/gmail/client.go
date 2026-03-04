package gmail

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	gmailapi "google.golang.org/api/gmail/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"

	"github.com/geekjourneyx/gcli/pkg/config"
	"github.com/geekjourneyx/gcli/pkg/errorsx"
	"github.com/geekjourneyx/gcli/pkg/model"
)

const (
	defaultLimit   = int64(20)
	maxRetry       = 3
	initialBackoff = 200 * time.Millisecond
)

type Client struct {
	svc *gmailapi.Service
}

type ListOptions struct {
	Label     string
	Query     string
	PageToken string
	Limit     int64
	Hydrate   bool
}

func NewClient(ctx context.Context, cfg config.RuntimeConfig, tokenSource oauth2.TokenSource, extraOpts ...option.ClientOption) (*Client, error) {
	opts := []option.ClientOption{
		option.WithTokenSource(tokenSource),
		option.WithScopes(gmailapi.GmailReadonlyScope),
	}
	if strings.TrimSpace(cfg.APIEndpoint) != "" {
		opts = append(opts, option.WithEndpoint(normalizeEndpoint(cfg.APIEndpoint)))
	}
	opts = append(opts, extraOpts...)

	svc, err := gmailapi.NewService(ctx, opts...)
	if err != nil {
		return nil, errorsx.Wrap(errorsx.CodeInternal, "create gmail service failed", false, err)
	}
	return &Client{svc: svc}, nil
}

func (c *Client) ListMessages(ctx context.Context, opts ListOptions) (model.MessagePage, error) {
	if opts.Limit <= 0 {
		opts.Limit = defaultLimit
	}

	call := c.svc.Users.Messages.List("me").MaxResults(opts.Limit)
	call = call.Fields("messages(id,threadId,snippet,internalDate)", "nextPageToken", "resultSizeEstimate")
	if strings.TrimSpace(opts.Label) != "" {
		call = call.LabelIds(opts.Label)
	}
	if strings.TrimSpace(opts.Query) != "" {
		call = call.Q(opts.Query)
	}
	if strings.TrimSpace(opts.PageToken) != "" {
		call = call.PageToken(opts.PageToken)
	}

	var resp *gmailapi.ListMessagesResponse
	if err := c.doWithRetry(ctx, "users.messages.list", func() error {
		var err error
		resp, err = call.Context(ctx).Do()
		return err
	}); err != nil {
		return model.MessagePage{}, err
	}

	page := model.MessagePage{
		Messages:           make([]model.MessageSummary, 0, len(resp.Messages)),
		NextPageToken:      resp.NextPageToken,
		ResultSizeEstimate: resp.ResultSizeEstimate,
	}

	for _, m := range resp.Messages {
		page.Messages = append(page.Messages, model.MessageSummary{
			ID:           m.Id,
			ThreadID:     m.ThreadId,
			Snippet:      m.Snippet,
			InternalDate: internalDateToRFC3339(m.InternalDate),
		})
	}

	if !opts.Hydrate {
		return page, nil
	}

	for i := range page.Messages {
		summary, err := c.getMessageSummary(ctx, page.Messages[i].ID)
		if err != nil {
			return model.MessagePage{}, err
		}
		// Keep list-level snippet/date if the detail API did not provide values.
		if summary.Snippet == "" {
			summary.Snippet = page.Messages[i].Snippet
		}
		if summary.InternalDate == "" {
			summary.InternalDate = page.Messages[i].InternalDate
		}
		page.Messages[i] = summary
	}

	return page, nil
}

func (c *Client) GetMessage(ctx context.Context, id, format string) (model.MessageDetail, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return model.MessageDetail{}, errorsx.New(errorsx.CodeInputInvalid, "message id is required", false)
	}

	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		format = "metadata"
	}
	if format != "metadata" && format != "full" && format != "minimal" && format != "raw" {
		return model.MessageDetail{}, errorsx.New(errorsx.CodeInputInvalid, "invalid format, expected metadata|full|minimal|raw", false)
	}

	call := c.svc.Users.Messages.Get("me", id).Format(format)
	if format == "metadata" {
		call = call.MetadataHeaders("From", "To", "Subject", "Date")
	}

	var msg *gmailapi.Message
	if err := c.doWithRetry(ctx, "users.messages.get", func() error {
		var err error
		msg, err = call.Context(ctx).Do()
		return err
	}); err != nil {
		return model.MessageDetail{}, err
	}

	return toMessageDetail(msg), nil
}

func (c *Client) getMessageSummary(ctx context.Context, id string) (model.MessageSummary, error) {
	call := c.svc.Users.Messages.Get("me", id).Format("metadata").MetadataHeaders("From", "Subject", "Date")

	var msg *gmailapi.Message
	if err := c.doWithRetry(ctx, "users.messages.get", func() error {
		var err error
		msg, err = call.Context(ctx).Do()
		return err
	}); err != nil {
		return model.MessageSummary{}, err
	}

	detail := toMessageDetail(msg)
	return model.MessageSummary{
		ID:           detail.ID,
		ThreadID:     detail.ThreadID,
		Snippet:      detail.Snippet,
		InternalDate: detail.InternalDate,
		From:         detail.Headers["from"],
		Subject:      detail.Headers["subject"],
	}, nil
}

func (c *Client) doWithRetry(ctx context.Context, opName string, op func() error) error {
	backoff := initialBackoff
	for i := 0; i < maxRetry; i++ {
		err := op()
		if err == nil {
			return nil
		}

		mapped := mapGoogleAPIError(err, opName)
		if !mapped.Retryable || i == maxRetry-1 {
			return mapped
		}

		t := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			t.Stop()
			return errorsx.From(ctx.Err())
		case <-t.C:
		}
		backoff *= 2
	}
	return errorsx.New(errorsx.CodeInternal, "retry loop exhausted unexpectedly", false)
}

func mapGoogleAPIError(err error, opName string) *errorsx.AppError {
	if err == nil {
		return nil
	}
	if appErr, ok := err.(*errorsx.AppError); ok {
		return appErr
	}
	if ctxErr := errorsx.From(err); ctxErr.Code == errorsx.CodeTimeout || ctxErr.Code == errorsx.CodeCanceled {
		return ctxErr
	}

	var apiErr *googleapi.Error
	if !errors.As(err, &apiErr) {
		return errorsx.Wrap(errorsx.CodeInternal, "gmail API call failed", false, err).
			AddDetail("operation", opName).
			AddDetail("source", "gmail_client")
	}

	reason := ""
	if len(apiErr.Errors) > 0 {
		reason = apiErr.Errors[0].Reason
	}

	switch {
	case apiErr.Code == http.StatusBadRequest:
		return errorsx.Wrap(errorsx.CodeInputInvalid, fmt.Sprintf("gmail bad request: %s", apiErr.Message), false, err).
			AddDetail("operation", opName).
			AddDetail("http_status", fmt.Sprintf("%d", apiErr.Code)).
			AddDetail("google_reason", reason)
	case apiErr.Code == http.StatusUnauthorized:
		return errorsx.Wrap(errorsx.CodeAuthRefreshFailed, "authentication failed: invalid or expired token", false, err).
			AddDetail("operation", opName).
			AddDetail("http_status", fmt.Sprintf("%d", apiErr.Code)).
			AddDetail("google_reason", reason)
	case apiErr.Code == http.StatusForbidden && reason == "insufficientPermissions":
		return errorsx.Wrap(errorsx.CodeAuthScopeInsufficient, "OAuth scope insufficient for requested operation", false, err).
			AddDetail("operation", opName).
			AddDetail("http_status", fmt.Sprintf("%d", apiErr.Code)).
			AddDetail("google_reason", reason)
	case apiErr.Code == http.StatusForbidden:
		return errorsx.Wrap(errorsx.CodeAuthRefreshFailed, "access forbidden by Gmail API", false, err).
			AddDetail("operation", opName).
			AddDetail("http_status", fmt.Sprintf("%d", apiErr.Code)).
			AddDetail("google_reason", reason)
	case apiErr.Code == http.StatusNotFound:
		return errorsx.Wrap(errorsx.CodeMailNotFound, "message not found", false, err).
			AddDetail("operation", opName).
			AddDetail("http_status", fmt.Sprintf("%d", apiErr.Code)).
			AddDetail("google_reason", reason)
	case apiErr.Code == http.StatusTooManyRequests:
		return errorsx.Wrap(errorsx.CodeGmailAPIQuota, "gmail API quota exceeded", true, err).
			AddDetail("operation", opName).
			AddDetail("http_status", fmt.Sprintf("%d", apiErr.Code)).
			AddDetail("google_reason", reason)
	case apiErr.Code >= 500:
		return errorsx.Wrap(errorsx.CodeGmailAPIUnavailable, "gmail API unavailable", true, err).
			AddDetail("operation", opName).
			AddDetail("http_status", fmt.Sprintf("%d", apiErr.Code)).
			AddDetail("google_reason", reason)
	default:
		return errorsx.Wrap(errorsx.CodeInternal, fmt.Sprintf("gmail API error: %s", apiErr.Message), false, err).
			AddDetail("operation", opName).
			AddDetail("http_status", fmt.Sprintf("%d", apiErr.Code)).
			AddDetail("google_reason", reason)
	}
}

func toMessageDetail(msg *gmailapi.Message) model.MessageDetail {
	headers := map[string]string{}
	var textParts []string
	var htmlParts []string
	if msg.Payload != nil {
		for _, h := range msg.Payload.Headers {
			name := strings.ToLower(strings.TrimSpace(h.Name))
			switch name {
			case "from", "to", "subject", "date":
				headers[name] = h.Value
			}
		}
		textParts, htmlParts = extractBodies(msg.Payload)
	}

	rawMIME := ""
	if strings.TrimSpace(msg.Raw) != "" {
		rawMIME = decodeBase64URL(msg.Raw)
	}

	return model.MessageDetail{
		ID:           msg.Id,
		ThreadID:     msg.ThreadId,
		LabelIDs:     msg.LabelIds,
		Snippet:      msg.Snippet,
		InternalDate: internalDateToRFC3339(msg.InternalDate),
		Headers:      headers,
		BodyText:     strings.Join(textParts, "\n\n"),
		BodyHTML:     strings.Join(htmlParts, "\n\n"),
		RawMIME:      rawMIME,
	}
}

func extractBodies(part *gmailapi.MessagePart) ([]string, []string) {
	if part == nil {
		return nil, nil
	}

	textParts := make([]string, 0, 2)
	htmlParts := make([]string, 0, 2)

	var walk func(p *gmailapi.MessagePart)
	walk = func(p *gmailapi.MessagePart) {
		if p == nil {
			return
		}

		mime := strings.ToLower(strings.TrimSpace(p.MimeType))
		body := ""
		if p.Body != nil && strings.TrimSpace(p.Body.Data) != "" {
			body = decodeBase64URL(p.Body.Data)
		}

		switch {
		case strings.HasPrefix(mime, "text/plain"):
			if strings.TrimSpace(body) != "" {
				textParts = append(textParts, body)
			}
		case strings.HasPrefix(mime, "text/html"):
			if strings.TrimSpace(body) != "" {
				htmlParts = append(htmlParts, body)
			}
		default:
			if len(p.Parts) == 0 && strings.TrimSpace(body) != "" && (mime == "" || strings.HasPrefix(mime, "text/")) {
				textParts = append(textParts, body)
			}
		}

		for _, child := range p.Parts {
			walk(child)
		}
	}

	walk(part)
	return textParts, htmlParts
}

func decodeBase64URL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	decoded, err := base64.RawURLEncoding.DecodeString(trimmed)
	if err == nil {
		return string(decoded)
	}

	decoded, err = base64.URLEncoding.DecodeString(trimmed)
	if err == nil {
		return string(decoded)
	}

	return ""
}

func internalDateToRFC3339(ms int64) string {
	if ms <= 0 {
		return ""
	}
	return time.UnixMilli(ms).UTC().Format(time.RFC3339)
}

func normalizeEndpoint(endpoint string) string {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		return ""
	}
	if strings.HasSuffix(trimmed, "/") {
		return trimmed
	}
	return trimmed + "/"
}
