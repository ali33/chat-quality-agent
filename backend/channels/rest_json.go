package channels

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// RestJSONCredentials configures HTTP JSON endpoints for sync (see channels/examples/).
type RestJSONCredentials struct {
	BaseURL string `json:"base_url"` // e.g. https://crm.example.com — no trailing slash required

	// ListConversationsPath — GET, query params appended: since (RFC3339), limit (int)
	ListConversationsPath string `json:"list_conversations_path"` // e.g. /api/cqa/conversations

	// MessagesPathTemplate — GET; must contain literal "{conversation_id}" replaced by external conv id.
	// Query param since (RFC3339) is appended.
	MessagesPathTemplate string `json:"messages_path_template"` // e.g. /api/cqa/conversations/{conversation_id}/messages

	// ExternalID — stable id for this source (stored on channel row; unique per tenant+type)
	ExternalID string `json:"external_id"`

	Headers            map[string]string `json:"headers,omitempty"`
	InsecureSkipVerify bool               `json:"insecure_skip_verify,omitempty"`
	TimeoutSeconds     int                `json:"timeout_seconds,omitempty"`
}

type restConvJSON struct {
	ExternalID       string                 `json:"external_id"`
	ExternalUserID   string                 `json:"external_user_id"`
	CustomerName     string                 `json:"customer_name"`
	LastMessageAt    string                 `json:"last_message_at"`
	Metadata         map[string]interface{} `json:"metadata"`
}

type restMsgJSON struct {
	ExternalID   string                   `json:"external_id"`
	SenderType   string                   `json:"sender_type"`
	SenderName   string                   `json:"sender_name"`
	Content      string                   `json:"content"`
	ContentType  string                   `json:"content_type"`
	SentAt       string                   `json:"sent_at"`
	Attachments  []Attachment             `json:"attachments"`
	RawData      map[string]interface{}   `json:"raw_data"`
}

// RestJSONAdapter pulls conversations and messages from your REST API (JSON).
type RestJSONAdapter struct {
	creds  RestJSONCredentials
	client *http.Client
}

// NewRestJSONAdapter validates credentials and returns an adapter.
func NewRestJSONAdapter(creds RestJSONCredentials) (*RestJSONAdapter, error) {
	if strings.TrimSpace(creds.BaseURL) == "" {
		return nil, fmt.Errorf("base_url is required")
	}
	if u, err := url.Parse(creds.BaseURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("base_url must be http or https URL")
	}
	if strings.TrimSpace(creds.ListConversationsPath) == "" {
		return nil, fmt.Errorf("list_conversations_path is required")
	}
	if strings.TrimSpace(creds.MessagesPathTemplate) == "" {
		return nil, fmt.Errorf("messages_path_template is required")
	}
	if !strings.Contains(creds.MessagesPathTemplate, "{conversation_id}") {
		return nil, fmt.Errorf("messages_path_template must contain {conversation_id}")
	}
	if strings.TrimSpace(creds.ExternalID) == "" {
		return nil, fmt.Errorf("external_id is required")
	}
	timeout := time.Duration(creds.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	var tr http.RoundTripper = http.DefaultTransport
	if t, ok := http.DefaultTransport.(*http.Transport); ok {
		cloned := t.Clone()
		if creds.InsecureSkipVerify {
			cloned.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // optional dev flag in credentials
		}
		tr = cloned
	}
	return &RestJSONAdapter{
		creds: creds,
		client: &http.Client{
			Timeout:   timeout,
			Transport: tr,
		},
	}, nil
}

func (a *RestJSONAdapter) doGET(ctx context.Context, rawURL string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, 0, err
	}
	for k, v := range a.creds.Headers {
		if strings.TrimSpace(k) != "" {
			req.Header.Set(k, v)
		}
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

func joinRESTURL(base, path string) string {
	base = strings.TrimRight(base, "/")
	path = strings.TrimSpace(path)
	if path == "" {
		return base
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}

func withQuery(u string, params map[string]string) string {
	if len(params) == 0 {
		return u
	}
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	sep := "?"
	if strings.Contains(u, "?") {
		sep = "&"
	}
	return u + sep + q.Encode()
}

func parseRESTTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	var lastErr error
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		} else {
			lastErr = err
		}
	}
	return time.Time{}, fmt.Errorf("parse time %q: %w", s, lastErr)
}

func decodeConversationsJSON(body []byte) ([]SyncedConversation, error) {
	var wrap struct {
		Conversations []restConvJSON `json:"conversations"`
	}
	if err := json.Unmarshal(body, &wrap); err == nil && wrap.Conversations != nil {
		return mapRestConversations(wrap.Conversations)
	}
	var arr []restConvJSON
	if err := json.Unmarshal(body, &arr); err == nil {
		return mapRestConversations(arr)
	}
	return nil, fmt.Errorf("conversations JSON: expected {\"conversations\":[...]} or a JSON array")
}

func mapRestConversations(in []restConvJSON) ([]SyncedConversation, error) {
	out := make([]SyncedConversation, 0, len(in))
	for _, c := range in {
		id := strings.TrimSpace(c.ExternalID)
		if id == "" {
			continue
		}
		var lastAt time.Time
		if c.LastMessageAt != "" {
			var err error
			lastAt, err = parseRESTTime(c.LastMessageAt)
			if err != nil {
				lastAt = time.Now()
			}
		} else {
			lastAt = time.Now()
		}
		meta := c.Metadata
		if meta == nil {
			meta = map[string]interface{}{}
		}
		out = append(out, SyncedConversation{
			ExternalID:     id,
			ExternalUserID: c.ExternalUserID,
			CustomerName:   c.CustomerName,
			LastMessageAt:  lastAt,
			Metadata:       meta,
		})
	}
	return out, nil
}

func decodeMessagesJSON(body []byte) ([]SyncedMessage, error) {
	var wrap struct {
		Messages []restMsgJSON `json:"messages"`
	}
	if err := json.Unmarshal(body, &wrap); err == nil && wrap.Messages != nil {
		return mapRestMessages(wrap.Messages)
	}
	var arr []restMsgJSON
	if err := json.Unmarshal(body, &arr); err == nil {
		return mapRestMessages(arr)
	}
	return nil, fmt.Errorf("messages JSON: expected {\"messages\":[...]} or a JSON array")
}

func mapRestMessages(in []restMsgJSON) ([]SyncedMessage, error) {
	out := make([]SyncedMessage, 0, len(in))
	for _, m := range in {
		id := strings.TrimSpace(m.ExternalID)
		if id == "" {
			continue
		}
		st := strings.TrimSpace(m.SenderType)
		if st == "" {
			st = "customer"
		}
		ct := strings.TrimSpace(m.ContentType)
		if ct == "" {
			ct = "text"
		}
		sent, err := parseRESTTime(m.SentAt)
		if err != nil {
			sent = time.Now()
		}
		raw := m.RawData
		if raw == nil {
			raw = map[string]interface{}{}
		}
		atts := m.Attachments
		if atts == nil {
			atts = nil
		}
		out = append(out, SyncedMessage{
			ExternalID:   id,
			SenderType:   st,
			SenderName:   m.SenderName,
			Content:      m.Content,
			ContentType:  ct,
			Attachments:  atts,
			SentAt:       sent,
			RawData:      raw,
		})
	}
	return out, nil
}

// FetchRecentConversations calls GET list_conversations_path?since=&limit=
func (a *RestJSONAdapter) FetchRecentConversations(ctx context.Context, since time.Time, limit int) ([]SyncedConversation, error) {
	if limit <= 0 {
		limit = 100
	}
	u := joinRESTURL(a.creds.BaseURL, a.creds.ListConversationsPath)
	u = withQuery(u, map[string]string{
		"since": since.UTC().Format(time.RFC3339),
		"limit": strconv.Itoa(limit),
	})
	body, code, err := a.doGET(ctx, u)
	if err != nil {
		return nil, err
	}
	if code < 200 || code >= 300 {
		return nil, fmt.Errorf("list conversations: HTTP %d: %s", code, truncateForErr(body))
	}
	return decodeConversationsJSON(body)
}

// FetchMessages calls GET messages_path with {conversation_id} replaced.
func (a *RestJSONAdapter) FetchMessages(ctx context.Context, conversationID string, since time.Time) ([]SyncedMessage, error) {
	path := strings.ReplaceAll(a.creds.MessagesPathTemplate, "{conversation_id}", conversationID)
	u := joinRESTURL(a.creds.BaseURL, path)
	u = withQuery(u, map[string]string{
		"since": since.UTC().Format(time.RFC3339),
	})
	body, code, err := a.doGET(ctx, u)
	if err != nil {
		return nil, err
	}
	if code < 200 || code >= 300 {
		return nil, fmt.Errorf("list messages: HTTP %d: %s", code, truncateForErr(body))
	}
	return decodeMessagesJSON(body)
}

func truncateForErr(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 500 {
		return s[:500] + "…"
	}
	return s
}

// HealthCheck performs a lightweight list request (limit 1).
func (a *RestJSONAdapter) HealthCheck(ctx context.Context) error {
	u := joinRESTURL(a.creds.BaseURL, a.creds.ListConversationsPath)
	u = withQuery(u, map[string]string{
		"since": time.Unix(0, 0).UTC().Format(time.RFC3339),
		"limit": "1",
	})
	body, code, err := a.doGET(ctx, u)
	if err != nil {
		return err
	}
	if code < 200 || code >= 300 {
		return fmt.Errorf("health check: HTTP %d: %s", code, truncateForErr(body))
	}
	_, err = decodeConversationsJSON(body)
	return err
}
