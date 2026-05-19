// Native amd64 reproduction for goccy/bigquery-emulator#468.
// Throwaway. Runs against an already-running emulator endpoint (os.Args[1]).
//
// Characterises the regression precisely:
//   1. COUNT(<col>) over an EMPTY table for every scalar column type
//      (is the NULL-on-empty regression type-dependent?).
//   2. The aggregate boundary on empty input: which aggregates are
//      spec-required to be 0 (COUNT/COUNTIF) vs legitimately NULL
//      (SUM/MIN/MAX/AVG/ANY_VALUE/LOGICAL_OR) — to pinpoint what the
//      regression actually breaks.
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
		{Name: "c_int", Type: bigquery.IntegerFieldType},
		{Name: "c_float", Type: bigquery.FloatFieldType},
		{Name: "c_num", Type: bigquery.NumericFieldType},
		{Name: "c_bool", Type: bigquery.BooleanFieldType},
		{Name: "c_str", Type: bigquery.StringFieldType},
		{Name: "c_bytes", Type: bigquery.BytesFieldType},
		{Name: "c_date", Type: bigquery.DateFieldType},
		{Name: "c_ts", Type: bigquery.TimestampFieldType},
	}
	if err := ds.Table("t").Create(ctx, &bigquery.TableMetadata{Schema: schema}); err != nil {
		log.Fatalf("Table.Create: %v", err)
	}
	tbl := fmt.Sprintf("`test.%s.t`", dsID)

	// scan into NullInt64/NullFloat64-agnostic: we only care NULL vs non-NULL,
	// so read the single column as bigquery.Value and report nil-ness.
	probe := func(label, sql string) {
		it, err := client.Query(sql).Read(ctx)
		if err != nil {
			fmt.Printf("%-42s | query error: %v\n", label, err)
			return
		}
		var row []bigquery.Value
		if err := it.Next(&row); err != nil && err != iterator.Done {
			fmt.Printf("%-42s | scan error: %v\n", label, err)
			return
		}
		if len(row) == 0 || row[0] == nil {
			fmt.Printf("%-42s | NULL\n", label)
		} else {
			fmt.Printf("%-42s | %v (%T)\n", label, row[0], row[0])
		}
	}

	fmt.Println("-- COUNT(<col>) over EMPTY table, per scalar type (expect INT64 0; never NULL) --")
	for _, col := range []string{"c_int", "c_float", "c_num", "c_bool", "c_str", "c_bytes", "c_date", "c_ts"} {
		probe("COUNT("+col+")", "SELECT COUNT("+col+") AS n FROM "+tbl)
	}
	probe("COUNT(*)", "SELECT COUNT(*) AS n FROM "+tbl)
	probe("COUNT(1)", "SELECT COUNT(1) AS n FROM "+tbl)
	probe("COUNTIF(TRUE)", "SELECT COUNTIF(TRUE) AS n FROM "+tbl)
	probe("COUNT(*) WHERE FALSE", "SELECT COUNT(*) AS n FROM "+tbl+" WHERE FALSE")

	fmt.Println("-- aggregate boundary over EMPTY table (only COUNT/COUNTIF are spec-required 0; rest are NULL) --")
	probe("SUM(c_int)", "SELECT SUM(c_int) AS n FROM "+tbl)
	probe("AVG(c_int)", "SELECT AVG(c_int) AS n FROM "+tbl)
	probe("MIN(c_int)", "SELECT MIN(c_int) AS n FROM "+tbl)
	probe("MAX(c_ts)", "SELECT MAX(c_ts) AS n FROM "+tbl)
	probe("ANY_VALUE(c_str)", "SELECT ANY_VALUE(c_str) AS n FROM "+tbl)
	probe("LOGICAL_OR(c_bool)", "SELECT LOGICAL_OR(c_bool) AS n FROM "+tbl)
}
