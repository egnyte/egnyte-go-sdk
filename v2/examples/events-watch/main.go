// Command events-watch tails the domain event stream: it obtains a
// cursor and then polls for new file system, note and permission events,
// printing each one. Egnyte asks pollers to wait at least five minutes
// between requests; this example uses a shorter interval for demo
// purposes — raise it for production use.
//
// Usage:
//
//	EGNYTE_DOMAIN=acme EGNYTE_TOKEN=... go run ./examples/events-watch
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	egnyte "github.com/egnyte/egnyte-go-sdk/v2"
)

func main() {
	ctx := context.Background()
	client, err := egnyte.NewClient(os.Getenv("EGNYTE_DOMAIN"),
		egnyte.WithToken(os.Getenv("EGNYTE_TOKEN")))
	if err != nil {
		log.Fatal(err)
	}

	cursor, _, err := client.Events.Cursor(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("starting at event %d (oldest retained: %d)\n",
		cursor.LatestEventID, cursor.OldestEventID)

	id := cursor.LatestEventID
	for {
		events, resp, err := client.Events.ListV2(ctx, id, nil)
		if err != nil {
			log.Fatal(err)
		}
		if resp.StatusCode == http.StatusNoContent {
			fmt.Println("no new events")
		}
		for _, e := range events.Events {
			switch e.Type {
			case "permission_change":
				fmt.Printf("[%d] %s changed permissions on %s\n", e.ID, e.Timestamp, e.Data.TargetPath)
			default:
				fmt.Printf("[%d] %s %s %s %s\n", e.ID, e.Timestamp, e.Type, e.Action, e.Data.TargetPath)
			}
		}
		if events.LatestID > 0 {
			id = events.LatestID // continue from the newest event seen
		}
		time.Sleep(30 * time.Second)
	}
}
