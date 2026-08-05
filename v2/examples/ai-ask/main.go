// Command ai-ask asks the Egnyte AI Assistant a question, optionally
// scoped to a folder, and polls for the answer. Requires a token with
// the Egnyte.ai scope; AI endpoints have separate, lower rate limits.
//
// Usage:
//
//	EGNYTE_DOMAIN=acme EGNYTE_TOKEN=... \
//	go run ./examples/ai-ask "What is our travel reimbursement policy?" [folder-id]
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
	if len(os.Args) < 2 {
		log.Fatalf("usage: %s <question> [folder-id]", os.Args[0])
	}
	question := os.Args[1]

	ctx := context.Background()
	client, err := egnyte.NewClient(os.Getenv("EGNYTE_DOMAIN"),
		egnyte.WithToken(os.Getenv("EGNYTE_TOKEN")))
	if err != nil {
		log.Fatal(err)
	}

	// Scope to a folder when given; otherwise search the whole domain.
	ask := egnyte.AssistantAskRequest{
		Question:         question,
		IncludeCitations: true,
	}
	if len(os.Args) > 2 {
		ask.SelectedItems = &egnyte.AISelectedItems{
			Folders: []egnyte.AIFolderRef{{ID: os.Args[2]}},
		}
	} else {
		ask.SelectedItems = &egnyte.AISelectedItems{AllEgnyteSearch: egnyte.Bool(true)}
	}

	// 1. Submit (async — returns an execution id).
	execution, _, err := client.AI.AskAssistant(ctx, ask)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("thinking (execution", execution.ExecutionID+")...")

	// 2. Poll until the answer is ready.
	for {
		status, _, err := client.AI.AssistantStatus(ctx, execution.ExecutionID, true)
		if err != nil {
			log.Fatal(err)
		}
		switch status.Status {
		case "COMPLETED":
			fmt.Println("\n" + status.ResponseText)
			if len(status.Citations) > 0 {
				fmt.Println("\nSources:")
				for _, c := range status.Citations {
					fmt.Printf("  - %s (%s)\n", c.Filename, c.EntryID)
				}
			}
			return
		case "FAILED":
			log.Fatal("assistant execution failed")
		}
		time.Sleep(3 * time.Second)
	}
}
