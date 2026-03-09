package lazy

import (
	"github.com/msjurset/golars/internal/expr"
)

// usedColumnsFromExprs collects all referenced column names from a list of expressions.
func usedColumnsFromExprs(exprs []expr.Expr) map[string]bool {
	cols := make(map[string]bool)
	for _, e := range exprs {
		for _, c := range e.UsedColumns() {
			cols[c] = true
		}
	}
	return cols
}

// pushdownProjection annotates Scan nodes with the minimal set of columns
// needed by their consumer. It walks top-down, collecting required columns
// from expressions in Select, WithColumns, and Filter nodes.
func pushdownProjection(plan *LogicalPlan) *LogicalPlan {
	if plan == nil {
		return nil
	}

	plan = clonePlan(plan)

	switch plan.nodeType {
	case NodeSelect:
		// Collect all columns referenced by the select expressions.
		needed := collectExprColumns(plan.selectExprs)
		plan.input = annotateProjection(plan.input, needed)
		return plan

	case NodeFilter:
		// Filter references columns in filterExpr but passes all through.
		// We can't prune here, just recurse.
		filterCols := plan.filterExpr.UsedColumns()
		plan.input = annotateProjectionAdditive(plan.input, filterCols)
		return plan

	case NodeWithColumns:
		// WithColumns keeps all existing columns and adds new ones.
		// We can't prune since all input columns pass through.
		exprCols := collectExprColumns(plan.selectExprs)
		plan.input = annotateProjectionAdditive(plan.input, exprCols)
		return plan
	}

	// For all other nodes, recurse into children.
	plan.input = pushdownProjection(plan.input)
	plan.inputR = pushdownProjection(plan.inputR)
	return plan
}

// annotateProjection pushes a known set of needed columns down through the plan
// tree, ultimately setting projectionCols on Scan nodes.
func annotateProjection(plan *LogicalPlan, needed []string) *LogicalPlan {
	if plan == nil || len(needed) == 0 {
		return plan
	}
	plan = clonePlan(plan)

	switch plan.nodeType {
	case NodeScan:
		plan.projectionCols = needed
		return plan

	case NodeScanCSV, NodeScanParquet:
		plan.scanProjection = needed
		return plan

	case NodeFilter:
		// Filter also needs its own columns.
		filterCols := plan.filterExpr.UsedColumns()
		combined := mergeStringSlices(needed, filterCols)
		plan.input = annotateProjection(plan.input, combined)
		return plan

	case NodeSelect:
		// Inner select determines what it needs from its own expressions.
		innerNeeded := collectExprColumns(plan.selectExprs)
		plan.input = annotateProjection(plan.input, innerNeeded)
		return plan

	case NodeWithColumns:
		exprCols := collectExprColumns(plan.selectExprs)
		combined := mergeStringSlices(needed, exprCols)
		plan.input = annotateProjection(plan.input, combined)
		return plan

	default:
		// For sort, groupby, join, etc., don't restrict projection further.
		plan.input = pushdownProjection(plan.input)
		plan.inputR = pushdownProjection(plan.inputR)
		return plan
	}
}

// annotateProjectionAdditive handles nodes where all input columns pass through
// (like Filter, WithColumns at the top level). We recurse normally since we
// cannot prune the full column set.
func annotateProjectionAdditive(plan *LogicalPlan, extraCols []string) *LogicalPlan {
	if plan == nil {
		return plan
	}
	_ = extraCols
	return pushdownProjection(plan)
}

// collectExprColumns gathers all column names referenced by a slice of expressions.
func collectExprColumns(exprs []expr.Expr) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, e := range exprs {
		for _, col := range e.UsedColumns() {
			if _, ok := seen[col]; !ok {
				seen[col] = struct{}{}
				result = append(result, col)
			}
		}
	}
	return result
}

// mergeStringSlices combines two string slices, deduplicating while preserving order.
func mergeStringSlices(a, b []string) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	result := make([]string, 0, len(a)+len(b))
	for _, s := range a {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			result = append(result, s)
		}
	}
	for _, s := range b {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			result = append(result, s)
		}
	}
	return result
}

// foldConstants evaluates constant sub-expressions at plan time, replacing
// them with literal values.
func foldConstants(plan *LogicalPlan) *LogicalPlan {
	switch plan.nodeType {
	case NodeSelect, NodeWithColumns:
		changed := false
		newExprs := make([]expr.Expr, len(plan.selectExprs))
		for i, e := range plan.selectExprs {
			folded := foldExpr(e)
			newExprs[i] = folded
			if folded != e {
				changed = true
			}
		}
		if changed {
			plan = clonePlan(plan)
			plan.selectExprs = newExprs
		}

	case NodeFilter:
		if plan.filterExpr != nil {
			folded := foldExpr(plan.filterExpr)
			if folded != plan.filterExpr {
				plan = clonePlan(plan)
				plan.filterExpr = folded
			}
		}
	}

	return plan
}

// foldExpr evaluates a constant expression and replaces it with a Lit.
// Non-constant expressions are returned unchanged.
func foldExpr(e expr.Expr) expr.Expr {
	if !e.IsConstant() {
		return e
	}

	// Evaluate with nil context (constants don't need a DataFrame).
	result, err := e.Evaluate(nil)
	if err != nil {
		return e
	}

	if result.Len() != 1 {
		return e
	}

	if result.IsNull(0) {
		return e
	}

	// Extract the scalar value and create a Lit expression.
	var val any
	switch result.DataType().String() {
	case "Int64":
		v, ok := result.GetInt64(0)
		if !ok {
			return e
		}
		val = v
	case "Float64":
		v, ok := result.GetFloat64(0)
		if !ok {
			return e
		}
		val = v
	case "Boolean":
		v, ok := result.GetBool(0)
		if !ok {
			return e
		}
		val = v
	case "String":
		v, ok := result.GetString(0)
		if !ok {
			return e
		}
		val = v
	default:
		return e
	}

	return expr.Lit(val)
}

// Optimize applies rule-based optimizations to the logical plan.
// Currently implements:
//   - Predicate pushdown: pushes filters below projections
//   - Projection merging: combines adjacent Select nodes
//   - Projection pushdown: pushes column projections down to scan nodes
//   - Constant folding: evaluates constant expressions at plan time
func Optimize(plan *LogicalPlan) *LogicalPlan {
	if plan == nil {
		return nil
	}

	// Apply optimization passes (fixed number to avoid infinite loops).
	result := plan
	for range 3 {
		result = applyRules(result)
	}

	// Projection pushdown runs as a final pass since it annotates scan nodes.
	result = pushdownProjection(result)

	return result
}

func applyRules(plan *LogicalPlan) *LogicalPlan {
	if plan == nil {
		return nil
	}

	// Recursively optimize children first (bottom-up).
	plan = clonePlan(plan)
	plan.input = applyRules(plan.input)
	plan.inputR = applyRules(plan.inputR)

	// Rule 1: Predicate pushdown — push Filter below Select/WithColumns/Rename/Drop.
	plan = pushdownPredicate(plan)

	// Rule 2: Projection merging — merge adjacent Select nodes.
	plan = mergeProjections(plan)

	// Rule 3: Constant folding — evaluate constant sub-expressions.
	plan = foldConstants(plan)

	// Rule 4: Common sub-expression elimination.
	plan = eliminateCommonSubexprs(plan)

	return plan
}

// eliminateCommonSubexprs deduplicates identical expressions in Select/WithColumns.
func eliminateCommonSubexprs(plan *LogicalPlan) *LogicalPlan {
	if plan.nodeType != NodeSelect && plan.nodeType != NodeWithColumns {
		return plan
	}

	if len(plan.selectExprs) <= 1 {
		return plan
	}

	// Hash expressions by their String() representation
	seen := make(map[string]int) // string -> first index
	deduped := make([]expr.Expr, 0, len(plan.selectExprs))
	changed := false

	for _, e := range plan.selectExprs {
		key := e.String()
		if _, exists := seen[key]; exists {
			changed = true
			continue // skip duplicate
		}
		seen[key] = len(deduped)
		deduped = append(deduped, e)
	}

	if !changed {
		return plan
	}

	result := clonePlan(plan)
	result.selectExprs = deduped
	return result
}

// pushdownPredicate pushes a Filter node below Select, WithColumns, Rename, or Drop nodes
// when the filter expression doesn't depend on newly computed columns.
func pushdownPredicate(plan *LogicalPlan) *LogicalPlan {
	if plan.nodeType != NodeFilter || plan.input == nil {
		return plan
	}

	child := plan.input
	switch child.nodeType {
	case NodeRename, NodeDrop:
		// Safe to push filter below rename/drop in most cases.
		// Swap: Filter(Rename(X)) -> Rename(Filter(X))
		newFilter := clonePlan(plan)
		newFilter.input = child.input
		newChild := clonePlan(child)
		newChild.input = newFilter
		return newChild

	case NodeSort:
		// Safe to push filter below sort.
		newFilter := clonePlan(plan)
		newFilter.input = child.input
		newChild := clonePlan(child)
		newChild.input = newFilter
		return newChild

	case NodeScanCSV, NodeScanParquet:
		// Push filter predicate into the scan node.
		if child.scanPredicate != nil {
			// Already has a predicate; leave the filter in place.
			return plan
		}
		newScan := clonePlan(child)
		newScan.scanPredicate = plan.filterExpr
		return newScan

	case NodeLimit:
		// NOT safe to push filter below limit (changes semantics).
		return plan

	default:
		return plan
	}
}

// mergeProjections combines two adjacent Select nodes into one.
func mergeProjections(plan *LogicalPlan) *LogicalPlan {
	if plan.nodeType != NodeSelect || plan.input == nil {
		return plan
	}
	if plan.input.nodeType == NodeSelect {
		// The outer Select can just replace the inner one since
		// it re-evaluates expressions against the input of the inner Select.
		merged := clonePlan(plan)
		merged.input = plan.input.input
		return merged
	}
	return plan
}

// clonePlan creates a shallow copy of a plan node.
func clonePlan(p *LogicalPlan) *LogicalPlan {
	if p == nil {
		return nil
	}
	cp := *p
	return &cp
}

// planEqual checks if two plans are structurally the same (by pointer identity
// of children, since we use clonePlan for any change).
func planEqual(a, b *LogicalPlan) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.nodeType != b.nodeType {
		return false
	}
	if a.input != b.input || a.inputR != b.inputR {
		return false
	}
	return true
}
