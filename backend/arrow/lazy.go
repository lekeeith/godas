package arrow

import (
	"fmt"

	"github.com/lekeeith/godas/core"
)

// LazyFrame is a lazy DataFrame that builds a query plan before execution.
type LazyFrame struct {
	source  *ArrowDataFrame
	ops     []lazyOp
	aliases map[string]string // column renames
}

type lazyOp struct {
	opType  string // "select", "filter", "withColumn", "groupBy", "sort", "limit"
	columns []string
	exprs   []*Expr
	mask    *Expr
	aggCols map[string]core.AggFunc
	groupBy []string
	sortBy  []string
	asc     []bool
	n       int
}

// Lazy creates a LazyFrame from a DataFrame.
func (df *ArrowDataFrame) Lazy() *LazyFrame {
	return &LazyFrame{source: df, aliases: make(map[string]string)}
}

// Select selects columns using expressions.
func (lf *LazyFrame) Select(exprs ...*Expr) *LazyFrame {
	newLf := lf.copy()
	newLf.ops = append(newLf.ops, lazyOp{opType: "select", exprs: exprs})
	return newLf
}

// Filter filters rows using a boolean expression.
func (lf *LazyFrame) Filter(expr *Expr) *LazyFrame {
	newLf := lf.copy()
	newLf.ops = append(newLf.ops, lazyOp{opType: "filter", mask: expr})
	return newLf
}

// WithColumn adds or replaces a column using an expression.
func (lf *LazyFrame) WithColumn(name string, expr *Expr) *LazyFrame {
	newLf := lf.copy()
	newLf.ops = append(newLf.ops, lazyOp{opType: "withColumn", exprs: []*Expr{expr}, columns: []string{name}})
	return newLf
}

// GroupBy sets grouping columns.
func (lf *LazyFrame) GroupBy(cols ...string) *LazyGroupBy {
	return &LazyGroupBy{lf: lf.copy(), cols: cols}
}

// Sort sorts by columns.
func (lf *LazyFrame) Sort(by []string, ascending []bool) *LazyFrame {
	newLf := lf.copy()
	newLf.ops = append(newLf.ops, lazyOp{opType: "sort", sortBy: by, asc: ascending})
	return newLf
}

// Limit limits the number of rows.
func (lf *LazyFrame) Limit(n int) *LazyFrame {
	newLf := lf.copy()
	newLf.ops = append(newLf.ops, lazyOp{opType: "limit", n: n})
	return newLf
}

// Collect executes the lazy plan and returns a DataFrame.
func (lf *LazyFrame) Collect() *ArrowDataFrame {
	df := lf.source

	for _, op := range lf.ops {
		switch op.opType {
		case "select":
			series := make([]*ArrowSeries, len(op.exprs))
			for i, expr := range op.exprs {
				result := expr.Eval(df)
				name := expr.alias
				if name == "" {
					name = exprName(expr)
				}
				series[i] = result.(*ArrowSeries).SetName(name).(*ArrowSeries)
			}
			df = NewDataFrame(series...)

		case "filter":
			mask := op.mask.Eval(df).(*ArrowSeries)
			boolMask := make([]bool, mask.Len())
			for i := 0; i < mask.Len(); i++ {
				if mask.NotNull(i) {
					boolMask[i] = mask.Bool(i)
				}
			}
			df = df.Filter(boolMask).(*ArrowDataFrame)

		case "withColumn":
			name := op.columns[0]
			result := op.exprs[0].Eval(df)
			df = df.WithColumn(name, result).(*ArrowDataFrame)

		case "sort":
			df = df.SortBy(op.sortBy, op.asc).(*ArrowDataFrame)

		case "limit":
			if df.Len() > op.n {
				df = df.Slice(0, op.n).(*ArrowDataFrame)
			}

		case "groupBy":
			// GroupBy + Agg combined
			result := df.Agg(op.groupBy, op.aggCols)
			df = result.(*ArrowDataFrame)
		}
	}

	return df
}

// copy creates a shallow copy of the LazyFrame.
func (lf *LazyFrame) copy() *LazyFrame {
	ops := make([]lazyOp, len(lf.ops))
	copy(ops, lf.ops)
	aliases := make(map[string]string)
	for k, v := range lf.aliases {
		aliases[k] = v
	}
	return &LazyFrame{source: lf.source, ops: ops, aliases: aliases}
}

// exprName returns a default name for an expression.
func exprName(e *Expr) string {
	switch e.op {
	case opCol:
		return e.col
	case opLit:
		return "_lit"
	case opAdd, opSub, opMul, opDiv, opMod:
		return fmt.Sprintf("%s_%s", exprName(e.left), exprName(e.right))
	case opAgg:
		return fmt.Sprintf("%s_%s", exprName(e.left), e.aggFn)
	case opApply:
		return fmt.Sprintf("%s_apply", exprName(e.left))
	default:
		return "_expr"
	}
}

// LazyGroupBy represents a grouped lazy frame waiting for aggregation.
type LazyGroupBy struct {
	lf   *LazyFrame
	cols []string
}

// Agg applies aggregation functions to the grouped lazy frame.
func (lgb *LazyGroupBy) Agg(aggs map[string]core.AggFunc) *LazyFrame {
	newLf := lgb.lf.copy()
	newLf.ops = append(newLf.ops, lazyOp{opType: "groupBy", groupBy: lgb.cols, aggCols: aggs})
	return newLf
}

// Describe returns a string representation of the query plan.
func (lf *LazyFrame) Describe() string {
	s := "LazyFrame Query Plan:\n"
	for i, op := range lf.ops {
		switch op.opType {
		case "select":
			s += fmt.Sprintf("  %d. Select: ", i)
			for j, e := range op.exprs {
				if j > 0 {
					s += ", "
				}
				s += e.String()
			}
			s += "\n"
		case "filter":
			s += fmt.Sprintf("  %d. Filter: %s\n", i, op.mask)
		case "withColumn":
			s += fmt.Sprintf("  %d. WithColumn: %s = %s\n", i, op.columns[0], op.exprs[0])
		case "sort":
			s += fmt.Sprintf("  %d. Sort: %v %v\n", i, op.sortBy, op.asc)
		case "limit":
			s += fmt.Sprintf("  %d. Limit: %d\n", i, op.n)
		case "groupBy":
			s += fmt.Sprintf("  %d. GroupBy: %v Agg: %v\n", i, op.groupBy, op.aggCols)
		}
	}
	return s
}
