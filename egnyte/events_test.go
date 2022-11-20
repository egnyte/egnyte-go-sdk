package egnyte

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

// Test Create folder
func TestEventCursor(t *testing.T) {
	client, err := NewClient(context.Background(), Config["domain"], Config["accessToken"], http.DefaultClient)
	if err != nil {
		t.Errorf("%s", err)
	}
	if client == nil {
		t.Errorf("%s", err)
	}
	event, err := client.EventCursor(context.Background())
	if err != nil {
		t.Errorf("%s", err)
	}
	if event == nil {
		t.Errorf("%s", err)
	}
	fmt.Println("latest event id", event.LatestEventID)
	fmt.Println("oldest event id", event.OldestEventID)
}
