// Command audit-report demonstrates the asynchronous audit workflow:
// it requests a login audit report for the last seven days, polls the
// job until completion, and prints the resulting events. Requires an
// administrator token with the Egnyte.audit scope.
//
// Usage:
//
//	EGNYTE_DOMAIN=acme EGNYTE_TOKEN=... go run ./examples/audit-report
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
	ctx := context.Background()
	client, err := egnyte.NewClient(os.Getenv("EGNYTE_DOMAIN"),
		egnyte.WithToken(os.Getenv("EGNYTE_TOKEN")))
	if err != nil {
		log.Fatal(err)
	}

	// 1. Kick off report generation (async — returns a job id).
	now := time.Now()
	jobID, _, err := client.Audit.CreateLoginsReport(ctx, egnyte.LoginAuditReportRequest{
		AuditReportRequest: egnyte.AuditReportRequest{
			Format:    "json",
			DateStart: now.AddDate(0, 0, -7).Format("2006-01-02"),
			DateEnd:   now.Format("2006-01-02"),
		},
		Events: []string{"logins", "failed_attempts"},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("report job started:", jobID)

	// 2. Poll until the job completes. Egnyte asks for at most one poll
	// every two minutes; shortened here for the demo.
	for {
		status, _, err := client.Audit.JobStatus(ctx, jobID)
		if err != nil {
			log.Fatal(err)
		}
		if status.Status == "completed" {
			fmt.Println("report ready at", status.ReportURL)
			break
		}
		fmt.Println("still running...")
		time.Sleep(15 * time.Second)
	}

	// 3. Fetch the report (JSON format supports offset/count paging).
	report, _, err := client.Audit.GetReport(ctx, "logins", jobID,
		&egnyte.AuditReportPageOptions{Count: 100})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d login events total; first page:\n", report.TotalCount)
	for _, event := range report.Events {
		fmt.Println(" ", event)
	}
}
