package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"httpserver/database"
	"httpserver/normalization"
)

func main() {
	dbPath := flag.String("db", "service.db", "Path to the service database (service.db by default)")
	projectID := flag.Int("project", 3, "Project ID to inspect")
	printJSON := flag.Bool("json", false, "Print raw JSON stats after the human-readable summary")
	flag.Parse()

	serviceDB, err := database.NewServiceDB(*dbPath)
	if err != nil {
		log.Fatalf("failed to open service database: %v", err)
	}
	defer serviceDB.Close()

	mapper := normalization.NewCounterpartyMapper(serviceDB)
	stats, err := mapper.GetNormalizedCounterpartyStats(*projectID)
	if err != nil {
		log.Fatalf("failed to collect stats: %v", err)
	}

	fmt.Println("\n--- Counterparty Normalization Statistics ---")
	fmt.Printf("Project ID: %d\n", stats.ProjectID)
	fmt.Printf("Generated At: %s\n", stats.GeneratedAt.Format(timeLayout))
	fmt.Printf("Total Mapped Records: %d\n", stats.TotalMappedCounterparties)
	fmt.Printf("Unique Normalized Names: %d\n", stats.UniqueNormalizedNames)
	fmt.Printf("Groups with Duplicates: %d\n", stats.GroupsWithDuplicates)
	fmt.Printf("Unmatched Records (no INN/BIN): %d\n", stats.UnmatchedRecords)

	fmt.Println("\nTop Duplicate Groups:")
	if len(stats.TopGroups) == 0 {
		fmt.Println("  No duplicate groups detected.")
	} else {
		for _, group := range stats.TopGroups {
			fmt.Printf("  - %s (%s): %d records\n", group.Identifier, group.KeyType, group.Count)
		}
	}

	if *printJSON {
		payload, err := json.MarshalIndent(stats, "", "  ")
		if err != nil {
			log.Fatalf("failed to marshal stats: %v", err)
		}
		fmt.Println("\nJSON payload:")
		fmt.Println(string(payload))
	}
}

const timeLayout = "2006-01-02 15:04:05"
