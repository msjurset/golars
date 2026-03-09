package expr

// mergeColumns combines two column name slices, deduplicating while preserving order.
func mergeColumns(a, b []string) []string {
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

// mergeColumnsMulti combines multiple column name slices.
func mergeColumnsMulti(slices ...[]string) []string {
	var result []string
	for _, s := range slices {
		result = mergeColumns(result, s)
	}
	return result
}

// --- colExpr ---

func (c *colExpr) UsedColumns() []string { return []string{c.name} }
func (c *colExpr) IsConstant() bool      { return false }

// --- litExpr ---

func (l *litExpr) UsedColumns() []string { return nil }
func (l *litExpr) IsConstant() bool      { return true }

// --- allColsExpr ---

func (a *allColsExpr) UsedColumns() []string { return nil }
func (a *allColsExpr) IsConstant() bool      { return false }

// --- aliasExpr ---

func (a *aliasExpr) UsedColumns() []string { return a.inner.UsedColumns() }
func (a *aliasExpr) IsConstant() bool      { return a.inner.IsConstant() }

// --- binaryExpr ---

func (b *binaryExpr) UsedColumns() []string {
	return mergeColumns(b.left.UsedColumns(), b.right.UsedColumns())
}
func (b *binaryExpr) IsConstant() bool { return b.left.IsConstant() && b.right.IsConstant() }

// --- comparisonExpr ---

func (c *comparisonExpr) UsedColumns() []string {
	return mergeColumns(c.left.UsedColumns(), c.right.UsedColumns())
}
func (c *comparisonExpr) IsConstant() bool { return c.left.IsConstant() && c.right.IsConstant() }

// --- logicalExpr ---

func (l *logicalExpr) UsedColumns() []string {
	return mergeColumns(l.left.UsedColumns(), l.right.UsedColumns())
}
func (l *logicalExpr) IsConstant() bool { return l.left.IsConstant() && l.right.IsConstant() }

// --- notExpr ---

func (n *notExpr) UsedColumns() []string { return n.inner.UsedColumns() }
func (n *notExpr) IsConstant() bool      { return n.inner.IsConstant() }

// --- aggExpr ---

func (a *aggExpr) UsedColumns() []string { return a.inner.UsedColumns() }
func (a *aggExpr) IsConstant() bool      { return false }

// --- quantileExpr ---

func (q *quantileExpr) UsedColumns() []string { return q.inner.UsedColumns() }
func (q *quantileExpr) IsConstant() bool      { return false }

// --- castExpr ---

func (e *castExpr) UsedColumns() []string { return e.inner.UsedColumns() }
func (e *castExpr) IsConstant() bool      { return e.inner.IsConstant() }

// --- isNullExpr ---

func (e *isNullExpr) UsedColumns() []string { return e.inner.UsedColumns() }
func (e *isNullExpr) IsConstant() bool      { return e.inner.IsConstant() }

// --- fillNullExpr ---

func (e *fillNullExpr) UsedColumns() []string {
	return mergeColumns(e.inner.UsedColumns(), e.fill.UsedColumns())
}
func (e *fillNullExpr) IsConstant() bool { return e.inner.IsConstant() && e.fill.IsConstant() }

// --- sortExpr ---

func (e *sortExpr) UsedColumns() []string { return e.inner.UsedColumns() }
func (e *sortExpr) IsConstant() bool      { return false }

// --- strContainsExpr ---

func (e *strContainsExpr) UsedColumns() []string { return e.inner.UsedColumns() }
func (e *strContainsExpr) IsConstant() bool      { return false }

// --- strTransformExpr ---

func (e *strTransformExpr) UsedColumns() []string { return e.inner.UsedColumns() }
func (e *strTransformExpr) IsConstant() bool      { return false }

// --- strLenExpr ---

func (e *strLenExpr) UsedColumns() []string { return e.inner.UsedColumns() }
func (e *strLenExpr) IsConstant() bool      { return false }

// --- strMethodExpr ---

func (e *strMethodExpr) UsedColumns() []string { return e.inner.UsedColumns() }
func (e *strMethodExpr) IsConstant() bool      { return false }

// --- windowExpr ---

func (w *windowExpr) UsedColumns() []string {
	cols := w.inner.UsedColumns()
	for _, name := range w.partitionBy {
		cols = mergeColumns(cols, []string{name})
	}
	return cols
}
func (w *windowExpr) IsConstant() bool { return false }

// --- whenExpr ---

func (w *whenExpr) UsedColumns() []string {
	cols := mergeColumns(w.condition.UsedColumns(), w.thenVal.UsedColumns())
	if w.otherwiseVal != nil {
		cols = mergeColumns(cols, w.otherwiseVal.UsedColumns())
	}
	return cols
}
func (w *whenExpr) IsConstant() bool {
	if w.otherwiseVal != nil {
		return w.condition.IsConstant() && w.thenVal.IsConstant() && w.otherwiseVal.IsConstant()
	}
	return w.condition.IsConstant() && w.thenVal.IsConstant()
}

// --- isInExpr ---

func (e *isInExpr) UsedColumns() []string { return e.inner.UsedColumns() }
func (e *isInExpr) IsConstant() bool      { return false }

// --- isBetweenExpr ---

func (e *isBetweenExpr) UsedColumns() []string {
	return mergeColumnsMulti(e.inner.UsedColumns(), e.lower.UsedColumns(), e.upper.UsedColumns())
}
func (e *isBetweenExpr) IsConstant() bool {
	return e.inner.IsConstant() && e.lower.IsConstant() && e.upper.IsConstant()
}

// --- rankExpr ---

func (e *rankExpr) UsedColumns() []string { return e.inner.UsedColumns() }
func (e *rankExpr) IsConstant() bool      { return false }

// --- dtComponentExpr ---

func (e *dtComponentExpr) UsedColumns() []string { return e.inner.UsedColumns() }
func (e *dtComponentExpr) IsConstant() bool      { return false }

// --- dtTruncateExpr ---

func (e *dtTruncateExpr) UsedColumns() []string { return e.inner.UsedColumns() }
func (e *dtTruncateExpr) IsConstant() bool      { return false }

// --- dtStrftimeExpr ---

func (e *dtStrftimeExpr) UsedColumns() []string { return e.inner.UsedColumns() }
func (e *dtStrftimeExpr) IsConstant() bool      { return false }

// --- dtOffsetExpr ---

func (e *dtOffsetExpr) UsedColumns() []string { return e.inner.UsedColumns() }
func (e *dtOffsetExpr) IsConstant() bool      { return false }

// --- tryCastExpr ---

func (e *tryCastExpr) UsedColumns() []string { return e.inner.UsedColumns() }
func (e *tryCastExpr) IsConstant() bool      { return e.inner.IsConstant() }

// --- nameTransformExpr ---

func (e *nameTransformExpr) UsedColumns() []string { return e.inner.UsedColumns() }
func (e *nameTransformExpr) IsConstant() bool      { return e.inner.IsConstant() }

// --- shiftExpr ---

func (e *shiftExpr) UsedColumns() []string { return e.inner.UsedColumns() }
func (e *shiftExpr) IsConstant() bool      { return false }

// --- diffExpr ---

func (e *diffExpr) UsedColumns() []string { return e.inner.UsedColumns() }
func (e *diffExpr) IsConstant() bool      { return false }

// --- pctChangeExpr ---

func (e *pctChangeExpr) UsedColumns() []string { return e.inner.UsedColumns() }
func (e *pctChangeExpr) IsConstant() bool      { return false }

// --- cumExpr ---

func (e *cumExpr) UsedColumns() []string { return e.inner.UsedColumns() }
func (e *cumExpr) IsConstant() bool      { return false }

// --- rollingExpr ---

func (e *rollingExpr) UsedColumns() []string { return e.inner.UsedColumns() }
func (e *rollingExpr) IsConstant() bool      { return false }

// --- mathExpr ---

func (e *mathExpr) UsedColumns() []string { return e.inner.UsedColumns() }
func (e *mathExpr) IsConstant() bool      { return e.inner.IsConstant() }

// --- roundExpr ---

func (e *roundExpr) UsedColumns() []string { return e.inner.UsedColumns() }
func (e *roundExpr) IsConstant() bool      { return e.inner.IsConstant() }

// --- headExpr ---

func (e *headExpr) UsedColumns() []string { return e.inner.UsedColumns() }
func (e *headExpr) IsConstant() bool      { return false }

// --- tailExpr ---

func (e *tailExpr) UsedColumns() []string { return e.inner.UsedColumns() }
func (e *tailExpr) IsConstant() bool      { return false }

// --- gatherExpr ---

func (e *gatherExpr) UsedColumns() []string { return e.inner.UsedColumns() }
func (e *gatherExpr) IsConstant() bool      { return false }

// --- uniqueExpr ---

func (e *uniqueExpr) UsedColumns() []string { return e.inner.UsedColumns() }
func (e *uniqueExpr) IsConstant() bool      { return false }

// --- sortByExpr ---

func (e *sortByExpr) UsedColumns() []string {
	return mergeColumns(e.inner.UsedColumns(), e.by.UsedColumns())
}
func (e *sortByExpr) IsConstant() bool { return false }

// --- dtEpochExpr ---

func (e *dtEpochExpr) UsedColumns() []string { return e.inner.UsedColumns() }
func (e *dtEpochExpr) IsConstant() bool      { return false }

// --- dtTotalSecondsExpr ---

func (e *dtTotalSecondsExpr) UsedColumns() []string { return e.inner.UsedColumns() }
func (e *dtTotalSecondsExpr) IsConstant() bool      { return false }
