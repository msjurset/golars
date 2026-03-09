package lazy

import (
	"context"
	"fmt"

	"github.com/msjurset/golars/internal/dataframe"
	"github.com/msjurset/golars/internal/expr"
	csvio "github.com/msjurset/golars/internal/io/csv"
	parquetio "github.com/msjurset/golars/internal/io/parquet"
	"github.com/msjurset/golars/internal/series"
)

// Execute walks the logical plan bottom-up and produces a DataFrame.
func Execute(plan *LogicalPlan) (*dataframe.DataFrame, error) {
	if plan == nil {
		return nil, fmt.Errorf("golars: lazy: nil plan")
	}

	switch plan.nodeType {
	case NodeScan:
		return executeScan(plan)
	case NodeFilter:
		return executeFilter(plan)
	case NodeSelect:
		return executeSelect(plan)
	case NodeWithColumns:
		return executeWithColumns(plan)
	case NodeSort:
		return executeSort(plan)
	case NodeGroupBy:
		return executeGroupBy(plan)
	case NodeJoin:
		return executeJoin(plan)
	case NodeLimit:
		return executeLimit(plan)
	case NodeUnique:
		return executeUnique(plan)
	case NodeDrop:
		return executeDrop(plan)
	case NodeRename:
		return executeRename(plan)
	case NodeScanCSV:
		return executeScanCSV(plan)
	case NodeScanParquet:
		return executeScanParquet(plan)
	default:
		return nil, fmt.Errorf("golars: lazy: unknown plan node type %d", plan.nodeType)
	}
}

func executeScan(plan *LogicalPlan) (*dataframe.DataFrame, error) {
	if plan.df == nil {
		return nil, fmt.Errorf("golars: lazy: scan node has no DataFrame")
	}
	if len(plan.projectionCols) > 0 {
		return plan.df.Select(plan.projectionCols...)
	}
	return plan.df, nil
}

func executeFilter(plan *LogicalPlan) (*dataframe.DataFrame, error) {
	input, err := Execute(plan.input)
	if err != nil {
		return nil, err
	}

	ctx := &expr.Context{DF: input}
	mask, err := plan.filterExpr.Evaluate(ctx)
	if err != nil {
		return nil, fmt.Errorf("golars: lazy: filter expression: %w", err)
	}

	return input.Filter(mask)
}

func executeSelect(plan *LogicalPlan) (*dataframe.DataFrame, error) {
	input, err := Execute(plan.input)
	if err != nil {
		return nil, err
	}

	ctx := &expr.Context{DF: input}
	cols := make([]*series.Series, len(plan.selectExprs))
	for i, e := range plan.selectExprs {
		s, err := e.Evaluate(ctx)
		if err != nil {
			return nil, fmt.Errorf("golars: lazy: select expression %d: %w", i, err)
		}
		cols[i] = s
	}

	return dataframe.New(cols...)
}

func executeWithColumns(plan *LogicalPlan) (*dataframe.DataFrame, error) {
	input, err := Execute(plan.input)
	if err != nil {
		return nil, err
	}

	ctx := &expr.Context{DF: input}
	newCols := make([]*series.Series, len(plan.selectExprs))
	for i, e := range plan.selectExprs {
		s, err := e.Evaluate(ctx)
		if err != nil {
			return nil, fmt.Errorf("golars: lazy: with_columns expression %d: %w", i, err)
		}
		newCols[i] = s
	}

	return input.WithColumns(newCols...)
}

func executeSort(plan *LogicalPlan) (*dataframe.DataFrame, error) {
	input, err := Execute(plan.input)
	if err != nil {
		return nil, err
	}

	if len(plan.sortCols) == 1 {
		desc := false
		if len(plan.sortDesc) > 0 {
			desc = plan.sortDesc[0]
		}
		return input.Sort(plan.sortCols[0], desc)
	}

	return input.SortBy(plan.sortCols, plan.sortDesc)
}

func executeGroupBy(plan *LogicalPlan) (*dataframe.DataFrame, error) {
	input, err := Execute(plan.input)
	if err != nil {
		return nil, err
	}

	grouped, err := input.GroupBy(plan.groupKeys...)
	if err != nil {
		return nil, err
	}

	if len(plan.groupExprs) > 0 {
		return grouped.AggExprs(plan.groupExprs...)
	}

	return grouped.Agg(plan.groupAggs)
}

func executeJoin(plan *LogicalPlan) (*dataframe.DataFrame, error) {
	left, err := Execute(plan.input)
	if err != nil {
		return nil, err
	}

	right, err := Execute(plan.inputR)
	if err != nil {
		return nil, err
	}

	return left.Join(right, plan.joinOn, plan.joinType)
}

func executeLimit(plan *LogicalPlan) (*dataframe.DataFrame, error) {
	input, err := Execute(plan.input)
	if err != nil {
		return nil, err
	}

	return input.Head(plan.limit), nil
}

func executeUnique(plan *LogicalPlan) (*dataframe.DataFrame, error) {
	input, err := Execute(plan.input)
	if err != nil {
		return nil, err
	}

	return input.Unique(plan.uniqueSub...)
}

func executeDrop(plan *LogicalPlan) (*dataframe.DataFrame, error) {
	input, err := Execute(plan.input)
	if err != nil {
		return nil, err
	}

	return input.Drop(plan.dropCols...)
}

func executeRename(plan *LogicalPlan) (*dataframe.DataFrame, error) {
	input, err := Execute(plan.input)
	if err != nil {
		return nil, err
	}

	return input.Rename(plan.renameOld, plan.renameNew)
}

func executeScanCSV(plan *LogicalPlan) (*dataframe.DataFrame, error) {
	opts := make([]csvio.ReadOption, len(plan.scanCSVOpts))
	copy(opts, plan.scanCSVOpts)

	// Apply pushed-down column projection.
	if len(plan.scanProjection) > 0 {
		opts = append(opts, csvio.WithColumns(plan.scanProjection...))
	}

	cols, err := csvio.ReadFile(plan.filePath, opts...)
	if err != nil {
		return nil, fmt.Errorf("golars: lazy: scan_csv: %w", err)
	}

	df, err := dataframe.New(cols...)
	if err != nil {
		return nil, err
	}

	// Apply pushed-down predicate.
	if plan.scanPredicate != nil {
		ctx := &expr.Context{DF: df}
		mask, err := plan.scanPredicate.Evaluate(ctx)
		if err != nil {
			return nil, fmt.Errorf("golars: lazy: scan_csv predicate: %w", err)
		}
		df, err = df.Filter(mask)
		if err != nil {
			return nil, err
		}
	}

	return df, nil
}

func executeScanParquet(plan *LogicalPlan) (*dataframe.DataFrame, error) {
	df, err := parquetio.ReadFile(plan.filePath)
	if err != nil {
		return nil, fmt.Errorf("golars: lazy: scan_parquet: %w", err)
	}

	// Apply pushed-down column projection.
	if len(plan.scanProjection) > 0 {
		df, err = df.Select(plan.scanProjection...)
		if err != nil {
			return nil, err
		}
	}

	// Apply pushed-down predicate.
	if plan.scanPredicate != nil {
		ctx := &expr.Context{DF: df}
		mask, err := plan.scanPredicate.Evaluate(ctx)
		if err != nil {
			return nil, fmt.Errorf("golars: lazy: scan_parquet predicate: %w", err)
		}
		df, err = df.Filter(mask)
		if err != nil {
			return nil, err
		}
	}

	return df, nil
}

// ExecuteWithContext walks the logical plan with cancellation checking.
func ExecuteWithContext(ctx context.Context, plan *LogicalPlan) (*dataframe.DataFrame, error) {
	if plan == nil {
		return nil, fmt.Errorf("golars: lazy: nil plan")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Execute children with context first.
	var input *dataframe.DataFrame
	var err error
	if plan.input != nil {
		input, err = ExecuteWithContext(ctx, plan.input)
		if err != nil {
			return nil, err
		}
	}

	// For binary nodes (join).
	var inputR *dataframe.DataFrame
	if plan.inputR != nil {
		inputR, err = ExecuteWithContext(ctx, plan.inputR)
		if err != nil {
			return nil, err
		}
	}

	// Check context again after children.
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Build a temporary plan with pre-computed inputs as scan nodes.
	tempPlan := clonePlan(plan)
	if input != nil {
		tempPlan.input = &LogicalPlan{nodeType: NodeScan, df: input}
	}
	if inputR != nil {
		tempPlan.inputR = &LogicalPlan{nodeType: NodeScan, df: inputR}
	}

	return Execute(tempPlan)
}
