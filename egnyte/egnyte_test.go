package egnyte

import (
	"context"
	"net/http"
	"testing"
)

// Test Create new egnyte client
func TestNewClient(t *testing.T) {

	client, err := NewClient(context.Background(), Config["domain"], Config["accessToken"], http.DefaultClient)
	if err != nil {
		t.Errorf("%s", err)
	}
	if client == nil {
		t.Errorf("%s", err)
	}
}
