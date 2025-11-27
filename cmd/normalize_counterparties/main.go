package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"httpserver/database"
	"httpserver/normalization"
)

func main() {
	dbPath := flag.String("db", "service.db", "Path to the service database (service.db by default)")
	projectID := flag.Int("project", 3, "Project ID to normalize")
	dryRun := flag.Bool("dry-run", false, "Analyze changes without writing them to the database")
	flag.Parse()

	serviceDB, err := database.NewServiceDB(*dbPath)
	if err != nil {
		log.Fatalf("failed to open service database: %v", err)
	}
	defer serviceDB.Close()

	mapper := normalization.NewCounterpartyMapper(serviceDB)
	summary, err := mapper.NormalizeNamesForProject(*projectID, *dryRun)
	if err != nil {
		log.Fatalf("failed to normalize names: %v", err)
	}

	fmt.Println("\n--- Counterparty Name Normalization ---")
	fmt.Printf("Project ID: %d\n", summary.ProjectID)
	fmt.Printf("Dry Run: %t\n", summary.DryRun)
	fmt.Printf("Total Records: %d\n", summary.TotalRecords)
	fmt.Printf("Updated Records: %d\n", summary.UpdatedRecords)
	fmt.Printf(" - Names adjusted: %d\n", summary.UpdatedNameCount)
	fmt.Printf(" - Legal forms adjusted: %d\n", summary.UpdatedLegalFormCount)
	fmt.Printf("Skipped (empty names): %d\n", summary.SkippedWithoutName)
	if !summary.DryRun {
		fmt.Printf("Applied Updates: %d\n", summary.AppliedUpdates)
	} else {
		fmt.Println("Applied Updates: 0 (dry run)")
	}
	fmt.Printf("Duration: %s\n", summary.Duration.Round(time.Millisecond))
}
