// Command users-list lists users visible to an administrator. Pass an
// optional username to request one exact SCIM match.
//
// The access token needs the Egnyte.user scope and the authenticated user
// must be a domain administrator.
//
// Usage:
//
//	EGNYTE_DOMAIN=acme EGNYTE_TOKEN=... go run ./examples/users-list
//	EGNYTE_DOMAIN=acme EGNYTE_TOKEN=... go run ./examples/users-list jsmith
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	egnyte "github.com/egnyte/egnyte-go-sdk/v2"
)

func main() {
	if len(os.Args) > 2 {
		log.Fatalf("usage: %s [username]", os.Args[0])
	}

	client, err := egnyte.NewClient(requiredEnv("EGNYTE_DOMAIN"),
		egnyte.WithToken(requiredEnv("EGNYTE_TOKEN")))
	if err != nil {
		log.Fatal(err)
	}

	opts := &egnyte.UserListOptions{Count: 20}
	if len(os.Args) == 2 {
		// %q produces the quoted string required by SCIM filter syntax.
		opts.Filter = fmt.Sprintf("userName eq %q", os.Args[1])
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	users, _, err := client.Users.List(ctx, opts)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("showing %d of %d users\n", len(users.Resources), users.TotalResults)
	for _, user := range users.Resources {
		name := user.Name.Formatted
		if name == "" {
			name = strings.TrimSpace(user.Name.GivenName + " " + user.Name.FamilyName)
		}
		fmt.Printf("%d\t%s\t%s\tactive=%t\trole=%s\n",
			user.ID, user.UserName, name, user.Active, user.Role)
	}
}

func requiredEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("set %s", name)
	}
	return value
}
