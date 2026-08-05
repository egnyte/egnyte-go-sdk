// Command quickstart authenticates against an Egnyte domain and lists
// the contents of /Shared.
//
// Usage:
//
//	EGNYTE_DOMAIN=acme EGNYTE_TOKEN=... go run ./examples/quickstart
//
// To obtain an access token with an internal application's API credentials,
// see examples/authentication.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	egnyte "github.com/egnyte/egnyte-go-sdk/v2"
)

func main() {
	domain, token := os.Getenv("EGNYTE_DOMAIN"), os.Getenv("EGNYTE_TOKEN")
	if domain == "" || token == "" {
		log.Fatal("set EGNYTE_DOMAIN and EGNYTE_TOKEN")
	}

	ctx := context.Background()
	client, err := egnyte.NewClient(domain, egnyte.WithToken(token))
	if err != nil {
		log.Fatal(err)
	}

	info, _, err := client.Tokens.UserInfo(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("authenticated as %s %s (%s)\n\n", info.FirstName, info.LastName, info.Username)

	folder, resp, err := client.FileSystem.Get(ctx, "/Shared", &egnyte.FileSystemGetOptions{
		ListContent: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range folder.Folders {
		fmt.Println("folder:", f.Path)
	}
	for _, f := range folder.Files {
		fmt.Printf("file:   %s (%d bytes)\n", f.Path, f.Size)
	}
	fmt.Printf("\nAPI quota: %d/%d calls used today\n", resp.Rate.QuotaCurrent, resp.Rate.QuotaAllotted)
}
