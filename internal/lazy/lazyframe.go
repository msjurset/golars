package lazy

import (
	"github.com/msjurset/golars/internal/dataframe"
	"github.com/msjurset/golars/internal/expr"
)

// LazyFrame represents a lazy computation over a DataFrame. Operations are
// recorded as a logical plan and only executed when Collect is called.
type LazyFrame struct {
	plan *LogicalPlan
}

// FromDataFrame creates a LazyFrame from an eager DataFrame.
func FromDataFrame(df *dataframe.DataFrame) *LazyFrame {
	return &LazyFrame{
		plan: &LogicalPlan{
			nodeType: NodeScan,
			df:       df,
		},
	}
}

// Filter adds a filter operation to the lazy plan.
func (lf *LazyFrame) Filter(e expr.Expr) *LazyFrame {
	return &LazyFrame{
		plan: &LogicalPlan{
			nodeType:   NodeFilter,
			input:      lf.plan,
			filterExpr: e,
		},
	}
}

// Select adds a projection operation to the lazy plan.
func (lf *LazyFrame) Select(exprs ...expr.Expr) *LazyFrame {
	return &LazyFrame{
		plan: &LogicalPlan{
			nodeType:    NodeSelect,
			input:       lf.plan,
			selectExprs: exprs,
		},
	}
}

// WithColumns adds new computed columns to the lazy plan.
func (lf *LazyFrame) WithColumns(exprs ...expr.Expr) *LazyFrame {
	return &LazyFrame{
		plan: &LogicalPlan{
			nodeType:    NodeWithColumns,
			input:       lf.plan,
			selectExprs: exprs,
		},
	}
}

// Sort adds a sort operation to the lazy plan.
func (lf *LazyFrame) Sort(column string, descending bool) *LazyFrame {
	return &LazyFrame{
		plan: &LogicalPlan{
			nodeType: NodeSort,
			input:    lf.plan,
			sortCols: []string{column},
			sortDesc: []bool{descending},
		},
	}
}

// SortBy adds a multi-column sort operation to the lazy plan.
func (lf *LazyFrame) SortBy(columns []string, descending []bool) *LazyFrame {
	return &LazyFrame{
		plan: &LogicalPlan{
			nodeType: NodeSort,
			input:    lf.plan,
			sortCols: columns,
			sortDesc: descending,
		},
	}
}

// GroupBy returns a LazyGroupBy for deferred aggregation.
func (lf *LazyFrame) GroupBy(keys ...string) *LazyGroupBy {
	return &LazyGroupBy{
		lf:   lf,
		keys: keys,
	}
}

// Join adds a join operation to the lazy plan.
func (lf *LazyFrame) Join(other *LazyFrame, on []string, how dataframe.JoinType) *LazyFrame {
	return &LazyFrame{
		plan: &LogicalPlan{
			nodeType: NodeJoin,
			input:    lf.plan,
			inputR:   other.plan,
			joinOn:   on,
			joinType: how,
		},
	}
}

// Head adds a limit operation to the lazy plan.
func (lf *LazyFrame) Head(n int) *LazyFrame {
	return &LazyFrame{
		plan: &LogicalPlan{
			nodeType: NodeLimit,
			input:    lf.plan,
			limit:    n,
		},
	}
}

// Unique adds a deduplication operation to the lazy plan.
func (lf *LazyFrame) Unique(subset ...string) *LazyFrame {
	return &LazyFrame{
		plan: &LogicalPlan{
			nodeType:  NodeUnique,
			input:     lf.plan,
			uniqueSub: subset,
		},
	}
}

// Drop adds a column drop operation to the lazy plan.
func (lf *LazyFrame) Drop(columns ...string) *LazyFrame {
	return &LazyFrame{
		plan: &LogicalPlan{
			nodeType: NodeDrop,
			input:    lf.plan,
			dropCols: columns,
		},
	}
}

// Rename adds a column rename operation to the lazy plan.
func (lf *LazyFrame) Rename(old, newName string) *LazyFrame {
	return &LazyFrame{
		plan: &LogicalPlan{
			nodeType:  NodeRename,
			input:     lf.plan,
			renameOld: old,
			renameNew: newName,
		},
	}
}

// Collect materializes the lazy plan, executing all deferred operations.
func (lf *LazyFrame) Collect() (*dataframe.DataFrame, error) {
	optimized := Optimize(lf.plan)
	return Execute(optimized)
}

// Explain returns a human-readable representation of the logical plan.
func (lf *LazyFrame) Explain() string {
	return lf.plan.String()
}

// ExplainOptimized returns a human-readable representation of the optimized plan.
func (lf *LazyFrame) ExplainOptimized() string {
	return Optimize(lf.plan).String()
}

// LazyGroupBy holds a deferred groupby operation.
type LazyGroupBy struct {
	lf   *LazyFrame
	keys []string
}

// Agg applies aggregation functions and returns a new LazyFrame.
func (g *LazyGroupBy) Agg(aggs map[string]dataframe.AggFunc) *LazyFrame {
	return &LazyFrame{
		plan: &LogicalPlan{
			nodeType:  NodeGroupBy,
			input:     g.lf.plan,
			groupKeys: g.keys,
			groupAggs: aggs,
		},
	}
}
