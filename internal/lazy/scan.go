package lazy

import (
	csvio "github.com/msjurset/golars/internal/io/csv"
)

// ScanCSV creates a LazyFrame that defers reading a CSV file until Collect.
func ScanCSV(path string, opts ...csvio.ReadOption) *LazyFrame {
	return &LazyFrame{
		plan: &LogicalPlan{
			nodeType:    NodeScanCSV,
			filePath:    path,
			scanCSVOpts: opts,
		},
	}
}

// ScanParquet creates a LazyFrame that defers reading a Parquet file until Collect.
func ScanParquet(path string) *LazyFrame {
	return &LazyFrame{
		plan: &LogicalPlan{
			nodeType: NodeScanParquet,
			filePath: path,
		},
	}
}
