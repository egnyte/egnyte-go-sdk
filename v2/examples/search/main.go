// Command search finds files and folders available to the authenticated user.
// The optional folder argument limits results to that subtree.
//
// The access token needs the Egnyte.filesystem scope.
//
// Usage:
//
//	EGNYTE_DOMAIN=acme EGNYTE_TOKEN=... \
//	go run ./examples/search "quarterly report"
//
//	EGNYTE_DOMAIN=acme EGNYTE_TOKEN=... \
//	go run ./examples/search "quarterly report" /Shared/Finance
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	egnyte "github.com/egnyte/egnyte-go-sdk/v2"
)

func main() {
	if len(os.Args) < 2 || len(os.Args) > 3 {
		log.Fatalf("usage: %s <query> [folder]", os.Args[0])
	}

	client, err := egnyte.NewClient(requiredEnv("EGNYTE_DOMAIN"),
		egnyte.WithToken(requiredEnv("EGNYTE_TOKEN")))
	if err != nil {
		log.Fatal(err)
	}

	wantSnippets := true
	opts := &egnyte.SearchOptions{
		Count:            10,
		SnippetRequested: &wantSnippets,
		SortBy:           "score",
		SortDirection:    "descending",
	}
	if len(os.Args) == 3 {
		opts.Folder = os.Args[2]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	results, _, err := client.Search.Search(ctx, os.Args[1], opts)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("showing %d of %d matches\n", len(results.Results), results.TotalCount)
	for _, result := range results.Results {
		kind := "file"
		if result.IsFolder {
			kind = "folder"
		}
		fmt.Printf("%s\t%s\n", kind, result.Path)
		if result.Snippet != "" {
			fmt.Printf("  %s\n", result.Snippet)
		}
	}
}

func requiredEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("set %s", name)
	}
	return value
}
