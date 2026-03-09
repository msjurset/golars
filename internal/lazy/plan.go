package lazy

import (
	"fmt"
	"strings"

	"github.com/msjurset/golars/internal/dataframe"
	"github.com/msjurset/golars/internal/expr"
	csvio "github.com/msjurset/golars/internal/io/csv"
)

// PlanNodeType identifies the kind of operation a LogicalPlan node represents.
type PlanNodeType int

const (
	NodeScan        PlanNodeType = iota
	NodeFilter
	NodeSelect
	NodeWithColumns
	NodeSort
	NodeGroupBy
	NodeJoin
	NodeLimit
	NodeUnique
	NodeDrop
	NodeRename
	NodeScanCSV
	NodeScanParquet
)

// LogicalPlan represents a node in the logical query plan DAG.
type LogicalPlan struct {
	nodeType PlanNodeType
	input    *LogicalPlan // single input (most nodes)
	inputR   *LogicalPlan // second input (join only)

	// Node-specific data
	df          *dataframe.DataFrame       // for Scan
	filterExpr  expr.Expr                  // for Filter
	selectExprs []expr.Expr                // for Select/WithColumns
	sortCols    []string                   // for Sort
	sortDesc    []bool                     // for Sort
	groupKeys   []string                      // for GroupBy
	groupAggs   map[string]dataframe.AggFunc // for GroupBy (map-based)
	groupExprs  []dataframe.GroupByExpr      // for GroupBy (expr-based)
	joinOn      []string                   // for Join
	joinType    dataframe.JoinType         // for Join
	limit       int                        // for Limit
	uniqueSub   []string                   // for Unique
	dropCols    []string                   // for Drop
	renameOld      string                     // for Rename
	renameNew      string                     // for Rename
	projectionCols []string                   // for Scan (projection pushdown)

	// Scan file nodes
	filePath       string             // for ScanCSV/ScanParquet
	scanCSVOpts    []csvio.ReadOption // for ScanCSV
	scanPredicate  expr.Expr          // pushed-down filter for scan nodes
	scanProjection []string           // pushed-down columns for scan nodes
}

// String returns a human-readable, indented representation of the plan tree.
func (p *LogicalPlan) String() string {
	return formatPlan(p, 0)
}

func formatPlan(p *LogicalPlan, depth int) string {
	if p == nil {
		return ""
	}

	indent := strings.Repeat("  ", depth)
	var line string

	switch p.nodeType {
	case NodeScan:
		h, w := 0, 0
		if p.df != nil {
			h, w = p.df.Shape()
		}
		if len(p.projectionCols) > 0 {
			line = fmt.Sprintf("%sSCAN [DataFrame: %dx%d, projection: %v]", indent, h, w, p.projectionCols)
		} else {
			line = fmt.Sprintf("%sSCAN [DataFrame: %dx%d]", indent, h, w)
		}
	case NodeFilter:
		exprStr := "<nil>"
		if p.filterExpr != nil {
			exprStr = p.filterExpr.String()
		}
		line = fmt.Sprintf("%sFILTER [expr: %s]", indent, exprStr)
	case NodeSelect:
		parts := make([]string, len(p.selectExprs))
		for i, e := range p.selectExprs {
			parts[i] = e.String()
		}
		line = fmt.Sprintf("%sSELECT [cols: %s]", indent, strings.Join(parts, ", "))
	case NodeWithColumns:
		parts := make([]string, len(p.selectExprs))
		for i, e := range p.selectExprs {
			parts[i] = e.String()
		}
		line = fmt.Sprintf("%sWITH_COLUMNS [exprs: %s]", indent, strings.Join(parts, ", "))
	case NodeSort:
		parts := make([]string, len(p.sortCols))
		for i, c := range p.sortCols {
			dir := "asc"
			if i < len(p.sortDesc) && p.sortDesc[i] {
				dir = "desc"
			}
			parts[i] = fmt.Sprintf("%q %s", c, dir)
		}
		line = fmt.Sprintf("%sSORT [%s]", indent, strings.Join(parts, ", "))
	case NodeGroupBy:
		if len(p.groupExprs) > 0 {
			line = fmt.Sprintf("%sGROUPBY [keys: %v, exprs: %d]", indent, p.groupKeys, len(p.groupExprs))
		} else {
			aggs := make([]string, 0, len(p.groupAggs))
			for col, fn := range p.groupAggs {
				aggs = append(aggs, fmt.Sprintf("%s(%s)", aggFuncName(fn), col))
			}
			line = fmt.Sprintf("%sGROUPBY [keys: %v, aggs: %s]", indent, p.groupKeys, strings.Join(aggs, ", "))
		}
	case NodeJoin:
		line = fmt.Sprintf("%sJOIN [on: %v, how: %s]", indent, p.joinOn, joinTypeName(p.joinType))
	case NodeLimit:
		line = fmt.Sprintf("%sLIMIT [n: %d]", indent, p.limit)
	case NodeUnique:
		line = fmt.Sprintf("%sUNIQUE [subset: %v]", indent, p.uniqueSub)
	case NodeDrop:
		line = fmt.Sprintf("%sDROP [cols: %v]", indent, p.dropCols)
	case NodeRename:
		line = fmt.Sprintf("%sRENAME [%q -> %q]", indent, p.renameOld, p.renameNew)
	case NodeScanCSV:
		line = fmt.Sprintf("%sSCAN_CSV [file: %s]", indent, p.filePath)
		if len(p.scanProjection) > 0 {
			line += fmt.Sprintf(" [proj: %v]", p.scanProjection)
		}
		if p.scanPredicate != nil {
			line += fmt.Sprintf(" [pred: %s]", p.scanPredicate.String())
		}
	case NodeScanParquet:
		line = fmt.Sprintf("%sSCAN_PARQUET [file: %s]", indent, p.filePath)
		if len(p.scanProjection) > 0 {
			line += fmt.Sprintf(" [proj: %v]", p.scanProjection)
		}
		if p.scanPredicate != nil {
			line += fmt.Sprintf(" [pred: %s]", p.scanPredicate.String())
		}
	default:
		line = fmt.Sprintf("%sUNKNOWN", indent)
	}

	var sb strings.Builder
	sb.WriteString(line)

	if p.input != nil {
		sb.WriteString("\n")
		sb.WriteString(formatPlan(p.input, depth+1))
	}
	if p.inputR != nil {
		sb.WriteString("\n")
		sb.WriteString(formatPlan(p.inputR, depth+1))
	}

	return sb.String()
}

func aggFuncName(fn dataframe.AggFunc) string {
	switch fn {
	case dataframe.AggSum:
		return "sum"
	case dataframe.AggMean:
		return "mean"
	case dataframe.AggMin:
		return "min"
	case dataframe.AggMax:
		return "max"
	case dataframe.AggCount:
		return "count"
	case dataframe.AggFirst:
		return "first"
	case dataframe.AggLast:
		return "last"
	default:
		return "unknown"
	}
}

func joinTypeName(jt dataframe.JoinType) string {
	switch jt {
	case dataframe.InnerJoin:
		return "inner"
	case dataframe.LeftJoin:
		return "left"
	case dataframe.RightJoin:
		return "right"
	case dataframe.FullJoin:
		return "full"
	case dataframe.SemiJoin:
		return "semi"
	case dataframe.AntiJoin:
		return "anti"
	case dataframe.CrossJoin:
		return "cross"
	default:
		return "unknown"
	}
}
