package lazy

import (
	"fmt"

	"github.com/msjurset/golars/internal/dataframe"
	"github.com/msjurset/golars/internal/expr"
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
	default:
		return nil, fmt.Errorf("golars: lazy: unknown plan node type %d", plan.nodeType)
	}
}

func executeScan(plan *LogicalPlan) (*dataframe.DataFrame, error) {
	if plan.df == nil {
		return nil, fmt.Errorf("golars: lazy: scan node has no DataFrame")
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
