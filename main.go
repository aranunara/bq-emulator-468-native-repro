// Native amd64 reproduction for goccy/bigquery-emulator#468.
// Throwaway. Runs against an already-running emulator endpoint (os.Args[1]).
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func main() {
	endpoint := os.Args[1] // e.g. http://localhost:9070
	ctx := context.Background()
	client, err := bigquery.NewClient(ctx, "test",
		option.WithEndpoint(endpoint),
		option.WithoutAuthentication(),
		option.WithScopes(bigquery.Scope),
	)
	if err != nil {
		log.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	dsID := fmt.Sprintf("probe_%d", time.Now().UnixMicro())
	ds := client.Dataset(dsID)
	if err := ds.Create(ctx, nil); err != nil {
		log.Fatalf("Dataset.Create: %v", err)
	}
	schema := bigquery.Schema{
		{Name: "data_date", Type: bigquery.DateFieldType},
		{Name: "name", Type: bigquery.StringFieldType},
	}
	if err := ds.Table("t").Create(ctx, &bigquery.TableMetadata{Schema: schema}); err != nil {
		log.Fatalf("Table.Create: %v", err)
	}

	tbl := fmt.Sprintf("`test.%s.t`", dsID)
	cases := []struct{ label, sql string }{
		{"COUNT(*) empty", "SELECT COUNT(*) AS n FROM " + tbl},
		{"COUNT(data_date) empty", "SELECT COUNT(data_date) AS n FROM " + tbl},
		{"COUNT(name) empty", "SELECT COUNT(name) AS n FROM " + tbl},
		{"COUNT(1) empty", "SELECT COUNT(1) AS n FROM " + tbl},
		{"COUNTIF(TRUE) empty", "SELECT COUNTIF(TRUE) AS n FROM " + tbl},
		{"SUM(1) empty", "SELECT SUM(1) AS n FROM " + tbl},
		{"COUNT(*) WHERE FALSE", "SELECT COUNT(*) AS n FROM " + tbl + " WHERE FALSE"},
		{"COUNT(data_date) WHERE FALSE", "SELECT COUNT(data_date) AS n FROM " + tbl + " WHERE FALSE"},
	}
	for _, c := range cases {
		it, err := client.Query(c.sql).Read(ctx)
		if err != nil {
			fmt.Printf("%-32s | query error: %v\n", c.label, err)
			continue
		}
		var row struct {
			N bigquery.NullInt64 `bigquery:"n"`
		}
		if err := it.Next(&row); err != nil && err != iterator.Done {
			fmt.Printf("%-32s | scan error: %v\n", c.label, err)
			continue
		}
		if row.N.Valid {
			fmt.Printf("%-32s | INT64 %d\n", c.label, row.N.Int64)
		} else {
			fmt.Printf("%-32s | NULL\n", c.label)
		}
	}
}
