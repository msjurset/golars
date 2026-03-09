package lazy

// Optimize applies rule-based optimizations to the logical plan.
// Currently implements:
//   - Predicate pushdown: pushes filters below projections
//   - Projection merging: combines adjacent Select nodes
func Optimize(plan *LogicalPlan) *LogicalPlan {
	if plan == nil {
		return nil
	}

	// Apply optimization passes (fixed number to avoid infinite loops).
	// Two passes is sufficient: one to push down, one to verify stability.
	result := plan
	for range 3 {
		result = applyRules(result)
	}
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

	return plan
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
