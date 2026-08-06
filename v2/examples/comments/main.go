// Command comments adds a comment to a file or folder, lists its comments,
// and removes the newly created comment before exiting.
//
// The access token needs the Egnyte.filesystem scope and write access to the
// target. This example intentionally cleans up the comment it creates.
//
// Usage:
//
//	EGNYTE_DOMAIN=acme EGNYTE_TOKEN=... \
//	go run ./examples/comments /Shared/Reports/report.pdf "Reviewed by the SDK"
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
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if len(os.Args) != 3 {
		return fmt.Errorf("usage: %s <remote-path> <comment>", os.Args[0])
	}
	path, message := os.Args[1], os.Args[2]

	client, err := egnyte.NewClient(requiredEnv("EGNYTE_DOMAIN"),
		egnyte.WithToken(requiredEnv("EGNYTE_TOKEN")))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	comment, _, err := client.Comments.Add(ctx, path, message)
	if err != nil {
		return err
	}
	fmt.Printf("added comment %s to %s\n", comment.ID, path)

	// Always try to remove the example comment, even if listing fails.
	defer func() {
		time.Sleep(600 * time.Millisecond)
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if _, err := client.Comments.Delete(cleanupCtx, comment.ID); err != nil {
			log.Printf("cleanup: delete comment %s: %v", comment.ID, err)
			return
		}
		fmt.Println("deleted example comment")
	}()

	// The default per-token rate limit is 2 calls/second.
	time.Sleep(600 * time.Millisecond)
	comments, _, err := client.Comments.List(ctx, &egnyte.CommentListOptions{
		File:  path,
		Count: 20,
	})
	if err != nil {
		return err
	}

	fmt.Printf("%d comments on %s\n", comments.TotalResults, path)
	for _, item := range comments.Notes {
		fmt.Printf("%s\t%s\t%s\n", item.ID, item.Username, item.Message)
	}
	return nil
}

func requiredEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("set %s", name)
	}
	return value
}
