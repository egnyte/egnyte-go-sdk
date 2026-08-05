package main

import (
	"context"
	"net/http"

	legacy "github.com/egnyte/egnyte-go-sdk/egnyte"
	egnytev2 "github.com/egnyte/egnyte-go-sdk/v2"
)

func main() {
	_, _ = legacy.NewClient(context.Background(), "acme.egnyte.com", "token", http.DefaultClient)
	_, _ = egnytev2.NewClient("acme", egnytev2.WithToken("token"))
}
