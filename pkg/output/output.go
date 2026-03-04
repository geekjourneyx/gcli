package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/geekjourneyx/gcli/pkg/errorsx"
	"github.com/geekjourneyx/gcli/pkg/model"
)

const schemaVersion = "v1"

type Format string

const (
	FormatJSON  Format = "json"
	FormatTable Format = "table"
)

type Options struct {
	Format Format
	Writer io.Writer
}

type envelope struct {
	Version string         `json:"version"`
	Data    any            `json:"data"`
	Error   *errorEnvelope `json:"error"`
}

type errorEnvelope struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Retryable bool              `json:"retryable"`
	Details   map[string]string `json:"details,omitempty"`
}

func ParseFormat(raw string) (Format, error) {
	switch Format(strings.ToLower(strings.TrimSpace(raw))) {
	case FormatJSON:
		return FormatJSON, nil
	case FormatTable:
		return FormatTable, nil
	default:
		return "", errorsx.New(errorsx.CodeInputInvalid, "invalid output format, expected json|table", false)
	}
}

func RenderSuccess(data any, opts Options) error {
	if opts.Writer == nil {
		return errorsx.New(errorsx.CodeInternal, "output writer is nil", false)
	}
	if opts.Format == "" {
		opts.Format = FormatJSON
	}

	switch opts.Format {
	case FormatJSON:
		return renderJSON(envelope{Version: schemaVersion, Data: data, Error: nil}, opts.Writer)
	case FormatTable:
		return renderTable(data, opts.Writer)
	default:
		return errorsx.New(errorsx.CodeInputInvalid, "unsupported output format", false)
	}
}

func RenderError(appErr *errorsx.AppError, opts Options) error {
	if appErr == nil {
		appErr = errorsx.New(errorsx.CodeInternal, "unknown error", false)
	}
	if opts.Writer == nil {
		return nil
	}
	if opts.Format == "" {
		opts.Format = FormatJSON
	}

	switch opts.Format {
	case FormatJSON:
		return renderJSON(envelope{
			Version: schemaVersion,
			Data:    nil,
			Error: &errorEnvelope{
				Code:      string(appErr.Code),
				Message:   appErr.Message,
				Retryable: appErr.Retryable,
				Details:   appErr.Details,
			},
		}, opts.Writer)
	case FormatTable:
		_, err := fmt.Fprintf(opts.Writer, "ERROR\t%s\t%s\n", appErr.Code, appErr.Message)
		return err
	default:
		return renderJSON(envelope{Version: schemaVersion, Data: nil, Error: &errorEnvelope{Code: string(appErr.Code), Message: appErr.Message, Retryable: appErr.Retryable, Details: appErr.Details}}, opts.Writer)
	}
}

func renderJSON(v any, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

func renderTable(data any, w io.Writer) error {
	tw := tabwriter.NewWriter(w, 0, 8, 2, ' ', 0)
	var err error

	switch v := data.(type) {
	case model.MessagePage:
		err = writeMessagePage(tw, v)
	case *model.MessagePage:
		if v == nil {
			return tw.Flush()
		}
		err = writeMessagePage(tw, *v)
	case model.MessageDetail:
		err = writeMessageDetail(tw, v)
	case *model.MessageDetail:
		if v == nil {
			return tw.Flush()
		}
		err = writeMessageDetail(tw, *v)
	case model.AuthLoginData:
		err = writeAuthData(tw, v)
	case *model.AuthLoginData:
		if v == nil {
			return tw.Flush()
		}
		err = writeAuthData(tw, *v)
	default:
		_, err = fmt.Fprintf(tw, "%+v\n", v)
	}

	if err != nil {
		return err
	}

	return tw.Flush()
}

func writeMessagePage(tw *tabwriter.Writer, page model.MessagePage) error {
	if _, err := fmt.Fprintln(tw, "ID\tTHREAD\tFROM\tSUBJECT\tDATE\tLABELS"); err != nil {
		return err
	}
	for _, m := range page.Messages {
		displayDate := strings.TrimSpace(m.Date)
		if displayDate == "" {
			displayDate = m.InternalDate
		}
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", m.ID, m.ThreadID, m.From, m.Subject, displayDate, strings.Join(m.LabelIDs, ",")); err != nil {
			return err
		}
	}
	if page.NextPageToken != "" {
		if _, err := fmt.Fprintf(tw, "\nNEXT_PAGE_TOKEN\t%s\n", page.NextPageToken); err != nil {
			return err
		}
	}
	return nil
}

func writeMessageDetail(tw *tabwriter.Writer, msg model.MessageDetail) error {
	if _, err := fmt.Fprintf(tw, "ID\t%s\n", msg.ID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(tw, "THREAD\t%s\n", msg.ThreadID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(tw, "DATE\t%s\n", msg.InternalDate); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(tw, "SNIPPET\t%s\n", msg.Snippet); err != nil {
		return err
	}
	for k, val := range msg.Headers {
		if _, err := fmt.Fprintf(tw, "%s\t%s\n", strings.ToUpper(k), val); err != nil {
			return err
		}
	}
	return nil
}

func writeAuthData(tw *tabwriter.Writer, data model.AuthLoginData) error {
	if _, err := fmt.Fprintf(tw, "REFRESH_TOKEN\t%s\n", data.RefreshToken); err != nil {
		return err
	}
	if data.Scope != "" {
		if _, err := fmt.Fprintf(tw, "SCOPE\t%s\n", data.Scope); err != nil {
			return err
		}
	}
	if data.ExpiresAt != "" {
		if _, err := fmt.Fprintf(tw, "EXPIRES_AT\t%s\n", data.ExpiresAt); err != nil {
			return err
		}
	}
	for k, v := range data.Env {
		if _, err := fmt.Fprintf(tw, "%s\t%s\n", k, v); err != nil {
			return err
		}
	}
	return nil
}
