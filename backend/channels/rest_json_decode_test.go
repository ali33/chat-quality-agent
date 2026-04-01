package channels

import (
	"testing"
	"time"
)

func TestDecodeConversationsJSON_object(t *testing.T) {
	body := []byte(`{"conversations":[{"external_id":"a","last_message_at":"2026-03-31T10:00:00Z"}]}`)
	out, err := decodeConversationsJSON(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].ExternalID != "a" {
		t.Fatalf("got %+v", out)
	}
	if out[0].LastMessageAt.UTC().Format(time.RFC3339) != "2026-03-31T10:00:00Z" {
		t.Fatalf("time: %v", out[0].LastMessageAt)
	}
}

func TestDecodeConversationsJSON_array(t *testing.T) {
	body := []byte(`[{"external_id":"b","customer_name":"X"}]`)
	out, err := decodeConversationsJSON(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].ExternalID != "b" || out[0].CustomerName != "X" {
		t.Fatalf("got %+v", out)
	}
}

func TestDecodeMessagesJSON(t *testing.T) {
	body := []byte(`{"messages":[{"external_id":"m1","sender_type":"agent","content":"hi","sent_at":"2026-03-31T12:00:00Z"}]}`)
	out, err := decodeMessagesJSON(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].ExternalID != "m1" || out[0].Content != "hi" || out[0].SenderType != "agent" {
		t.Fatalf("got %+v", out)
	}
}

func TestNewRestJSONAdapter_validation(t *testing.T) {
	_, err := NewRestJSONAdapter(RestJSONCredentials{})
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = NewRestJSONAdapter(RestJSONCredentials{
		BaseURL:               "https://x.com",
		ListConversationsPath: "/l",
		MessagesPathTemplate:  "/m/{conversation_id}/x",
		ExternalID:            "e",
	})
	if err != nil {
		t.Fatal(err)
	}
}
