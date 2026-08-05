// Command upload-download uploads a local file to Egnyte, shares it via
// a domain-restricted link, downloads it back, and prints the link URL.
//
// Usage:
//
//	EGNYTE_DOMAIN=acme EGNYTE_TOKEN=... \
//	go run ./examples/upload-download <local-file> </Shared/target/path>
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	egnyte "github.com/egnyte/egnyte-go-sdk/v2"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("usage: %s <local-file> </Shared/target/path>", os.Args[0])
	}
	localPath, remotePath := os.Args[1], os.Args[2]

	ctx := context.Background()
	client, err := egnyte.NewClient(os.Getenv("EGNYTE_DOMAIN"),
		egnyte.WithToken(os.Getenv("EGNYTE_TOKEN")))
	if err != nil {
		log.Fatal(err)
	}

	// Upload — the reader is streamed, never buffered in memory.
	f, err := os.Open(localPath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	uploaded, _, err := client.FileSystemContent.Upload(ctx, remotePath, f, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("uploaded %s (entry %s, checksum %.16s...)\n", remotePath, uploaded.EntryID, uploaded.Checksum)

	// The default per-token rate limit is 2 calls/second; pace
	// back-to-back calls to stay under it.
	pace := func() { time.Sleep(600 * time.Millisecond) }
	pace()

	// Share it inside the domain.
	link, _, err := client.Links.Create(ctx, egnyte.CreateLinkRequest{
		Path: remotePath, Type: "file", Accessibility: "domain",
	})
	if err != nil {
		log.Fatal(err)
	}
	if len(link.Links) == 0 {
		log.Fatal("link created but none returned")
	}
	fmt.Println("share link:", link.Links[0].URL)
	pace()

	// Download it back to verify.
	body, _, err := client.FileSystemContent.Download(ctx, remotePath, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer body.Close()
	n, err := io.Copy(io.Discard, body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("downloaded %d bytes back\n", n)
}
