package model

// MessageSummary is a compact message representation for list/search responses.
type MessageSummary struct {
	ID           string   `json:"id"`
	ThreadID     string   `json:"thread_id"`
	Snippet      string   `json:"snippet"`
	InternalDate string   `json:"internal_date"`
	Date         string   `json:"date,omitempty"`
	From         string   `json:"from"`
	Subject      string   `json:"subject"`
	LabelIDs     []string `json:"label_ids,omitempty"`
}

// MessagePage is a paginated collection of message summaries.
type MessagePage struct {
	Messages           []MessageSummary `json:"messages"`
	NextPageToken      string           `json:"next_page_token,omitempty"`
	ResultSizeEstimate int64            `json:"result_size_estimate"`
}

// MessageDetail is a detailed representation of a single message.
type MessageDetail struct {
	ID           string            `json:"id"`
	ThreadID     string            `json:"thread_id"`
	LabelIDs     []string          `json:"label_ids,omitempty"`
	Snippet      string            `json:"snippet"`
	InternalDate string            `json:"internal_date"`
	Headers      map[string]string `json:"headers"`
	BodyText     string            `json:"body_text,omitempty"`
	BodyHTML     string            `json:"body_html,omitempty"`
	RawMIME      string            `json:"raw_mime,omitempty"`
}

// AuthLoginData is returned by auth login.
type AuthLoginData struct {
	RefreshToken string            `json:"refresh_token"`
	AccessToken  string            `json:"access_token,omitempty"`
	TokenType    string            `json:"token_type,omitempty"`
	Scope        string            `json:"scope,omitempty"`
	ExpiresAt    string            `json:"expires_at,omitempty"`
	Env          map[string]string `json:"env,omitempty"`
}
