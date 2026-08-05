// Command authentication obtains an OAuth access token for an internal
// application and demonstrates using and restoring it without printing it.
//
// Usage:
//
//	EGNYTE_DOMAIN=acme \
//	EGNYTE_KEY=... \
//	EGNYTE_SECRET=... \
//	EGNYTE_USERNAME=... \
//	EGNYTE_PASSWORD=... \
//	EGNYTE_SCOPES="Egnyte.filesystem Egnyte.link" \
//	go run ./examples/authentication
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/egnyte/egnyte-go-sdk/v2"
)

func main() {
	domain := requiredEnv("EGNYTE_DOMAIN")
	key := requiredEnv("EGNYTE_KEY")
	secret := requiredEnv("EGNYTE_SECRET")
	username := requiredEnv("EGNYTE_USERNAME")
	password := requiredEnv("EGNYTE_PASSWORD")

	scopes := strings.Fields(os.Getenv("EGNYTE_SCOPES"))
	if len(scopes) == 0 {
		scopes = []string{"Egnyte.filesystem"}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := egnyte.NewClient(domain)
	if err != nil {
		log.Fatal(err)
	}

	token, _, err := client.RequestToken(ctx, egnyte.PasswordCredentials{
		ClientID:     key,
		ClientSecret: secret,
		Username:     username,
		Password:     password,
		Scopes:       scopes,
	})
	if err != nil {
		log.Fatal(err)
	}

	// RequestToken installed token.AccessToken on client, so it is ready for
	// authenticated API calls. Never print access or refresh tokens.
	info, _, err := client.Tokens.UserInfo(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("authenticated as %s (%s)\n", info.Username, info.Email)
	fmt.Printf("received %s access token, valid for %s\n",
		token.TokenType, time.Duration(token.ExpiresIn)*time.Second)

	// A real application should save token.AccessToken and token.RefreshToken
	// in its secret manager or encrypted credential store. A later process can
	// restore the cached access token with WithToken:
	restored, err := egnyte.NewClient(domain, egnyte.WithToken(token.AccessToken))
	if err != nil {
		log.Fatal(err)
	}
	if _, _, err := restored.Tokens.UserInfo(ctx); err != nil {
		log.Fatal(err)
	}
	fmt.Println("cached-token restoration verified")
}

func requiredEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("set %s", name)
	}
	return value
}
