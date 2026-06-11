package arrow

import (
	"fmt"

	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/lekeeith/godas/core"
)

// Expr represents a composable, lazy expression that describes a transformation
// without immediately executing it. Inspired by Polars' expression system.
type Expr struct {
	// Op describes the operation type
	op exprOp
	// Column is the target column name
	col string
	// Value is a literal value for scalar operations
	val interface{}
	// Left/Right are sub-expressions for binary operations
	left, right *Expr
	// Fn is a custom function for Apply operations
	fn func(float64) float64
	// Alias renames the output
	alias string
	// AggFunc for aggregation expressions
	aggFn core.AggFunc
}

type exprOp int

const (
	opCol      exprOp = iota // Reference a column
	opLit                    // Literal value
	opAdd                    // a + b
	opSub                    // a - b
	opMul                    // a * b
	opDiv                    // a / b
	opMod                    // a % b
	opNeg                    // -a
	opAbs                    // abs(a)
	opEq                     // a == b
	opNe                     // a != b
	opLt                     // a < b
	opLe                     // a <= b
	opGt                     // a > b
	opGe                     // a >= b
	opAnd                    // a && b
	opOr                     // a || b
	opNot                    // !a
	opApply                  // custom function
	opAgg                    // aggregation
	opIsNull                 // is null
	opNotNull                // is not null
	opFillNA                 // fill null
	opCast                   // type cast
	opClip                   // clip values
	opStringContains         // string contains
	opStringReplace          // string replace
	opStringUpper            // string upper
	opStringLower            // string lower
)

// Col creates a column reference expression.
func Col(name string) *Expr {
	return &Expr{op: opCol, col: name}
}

// Lit creates a literal value expression.
func Lit(v interface{}) *Expr {
	return &Expr{op: opLit, val: v}
}

// --- Arithmetic ---

func (e *Expr) Add(other *Expr) *Expr {
	return &Expr{op: opAdd, left: e, right: other}
}

func (e *Expr) Sub(other *Expr) *Expr {
	return &Expr{op: opSub, left: e, right: other}
}

func (e *Expr) Mul(other *Expr) *Expr {
	return &Expr{op: opMul, left: e, right: other}
}

func (e *Expr) Div(other *Expr) *Expr {
	return &Expr{op: opDiv, left: e, right: other}
}

func (e *Expr) Mod(other *Expr) *Expr {
	return &Expr{op: opMod, left: e, right: other}
}

func (e *Expr) Neg() *Expr {
	return &Expr{op: opNeg, left: e}
}

func (e *Expr) Abs() *Expr {
	return &Expr{op: opAbs, left: e}
}

// --- Comparison ---

func (e *Expr) Eq(other *Expr) *Expr {
	return &Expr{op: opEq, left: e, right: other}
}

func (e *Expr) Ne(other *Expr) *Expr {
	return &Expr{op: opNe, left: e, right: other}
}

func (e *Expr) Lt(other *Expr) *Expr {
	return &Expr{op: opLt, left: e, right: other}
}

func (e *Expr) Le(other *Expr) *Expr {
	return &Expr{op: opLe, left: e, right: other}
}

func (e *Expr) Gt(other *Expr) *Expr {
	return &Expr{op: opGt, left: e, right: other}
}

func (e *Expr) Ge(other *Expr) *Expr {
	return &Expr{op: opGe, left: e, right: other}
}

// --- Logic ---

func (e *Expr) And(other *Expr) *Expr {
	return &Expr{op: opAnd, left: e, right: other}
}

func (e *Expr) Or(other *Expr) *Expr {
	return &Expr{op: opOr, left: e, right: other}
}

func (e *Expr) Not() *Expr {
	return &Expr{op: opNot, left: e}
}

// --- Null handling ---

func (e *Expr) IsNull() *Expr {
	return &Expr{op: opIsNull, left: e}
}

func (e *Expr) IsNotNull() *Expr {
	return &Expr{op: opNotNull, left: e}
}

func (e *Expr) FillNA(v interface{}) *Expr {
	return &Expr{op: opFillNA, left: e, val: v}
}

// --- Transform ---

func (e *Expr) Apply(fn func(float64) float64) *Expr {
	return &Expr{op: opApply, left: e, fn: fn}
}

func (e *Expr) Cast(dt core.DType) *Expr {
	return &Expr{op: opCast, left: e, val: dt}
}

func (e *Expr) Clip(lo, hi float64) *Expr {
	return &Expr{op: opClip, left: e, val: [2]float64{lo, hi}}
}

// --- Aggregation ---

func (e *Expr) Sum() *Expr     { return &Expr{op: opAgg, left: e, aggFn: core.AggSum} }
func (e *Expr) Mean() *Expr    { return &Expr{op: opAgg, left: e, aggFn: core.AggMean} }
func (e *Expr) Min() *Expr     { return &Expr{op: opAgg, left: e, aggFn: core.AggMin} }
func (e *Expr) Max() *Expr     { return &Expr{op: opAgg, left: e, aggFn: core.AggMax} }
func (e *Expr) Count() *Expr   { return &Expr{op: opAgg, left: e, aggFn: core.AggCount} }
func (e *Expr) Std() *Expr     { return &Expr{op: opAgg, left: e, aggFn: core.AggStd} }
func (e *Expr) Median() *Expr  { return &Expr{op: opAgg, left: e, aggFn: core.AggMedian} }
func (e *Expr) First() *Expr   { return &Expr{op: opAgg, left: e, aggFn: core.AggFirst} }
func (e *Expr) Last() *Expr    { return &Expr{op: opAgg, left: e, aggFn: core.AggLast} }
func (e *Expr) NUnique() *Expr { return &Expr{op: opAgg, left: e, aggFn: core.AggNUnique} }

// --- String ---

func (e *Expr) StrContains(pattern string) *Expr {
	return &Expr{op: opStringContains, left: e, val: pattern}
}

func (e *Expr) StrReplace(old, new string) *Expr {
	return &Expr{op: opStringReplace, left: e, val: [2]string{old, new}}
}

func (e *Expr) StrUpper() *Expr { return &Expr{op: opStringUpper, left: e} }
func (e *Expr) StrLower() *Expr { return &Expr{op: opStringLower, left: e} }

// --- Alias ---

func (e *Expr) Alias(name string) *Expr {
	e.alias = name
	return e
}

// String returns a human-readable representation of the expression.
func (e *Expr) String() string {
	switch e.op {
	case opCol:
		return fmt.Sprintf("col(%q)", e.col)
	case opLit:
		return fmt.Sprintf("lit(%v)", e.val)
	case opAdd:
		return fmt.Sprintf("(%s + %s)", e.left, e.right)
	case opSub:
		return fmt.Sprintf("(%s - %s)", e.left, e.right)
	case opMul:
		return fmt.Sprintf("(%s * %s)", e.left, e.right)
	case opDiv:
		return fmt.Sprintf("(%s / %s)", e.left, e.right)
	case opNeg:
		return fmt.Sprintf("(-%s)", e.left)
	case opAbs:
		return fmt.Sprintf("abs(%s)", e.left)
	case opEq:
		return fmt.Sprintf("(%s == %s)", e.left, e.right)
	case opGt:
		return fmt.Sprintf("(%s > %s)", e.left, e.right)
	case opAgg:
		return fmt.Sprintf("%s(%s)", e.aggFn, e.left)
	case opApply:
		return fmt.Sprintf("apply(%s)", e.left)
	default:
		return fmt.Sprintf("expr(op=%d)", e.op)
	}
}

// Eval evaluates the expression against a DataFrame, returning a Series.
func (e *Expr) Eval(df *ArrowDataFrame) core.Series {
	switch e.op {
	case opCol:
		return df.Col(e.col)

	case opLit:
		// Create a constant series
		rows, _ := df.Shape()
		switch v := e.val.(type) {
		case float64:
			return newConstFloat64Series("_lit", v, rows)
		case int:
			return newConstFloat64Series("_lit", float64(v), rows)
		case int64:
			return newConstFloat64Series("_lit", float64(v), rows)
		case string:
			return newConstStringSeries("_lit", v, rows)
		case bool:
			return newConstBoolSeries("_lit", v, rows)
		default:
			return newConstFloat64Series("_lit", 0, rows)
		}

	case opAdd:
		return e.left.Eval(df).(*ArrowSeries).Add(e.right.Eval(df))
	case opSub:
		return e.left.Eval(df).(*ArrowSeries).Sub(e.right.Eval(df))
	case opMul:
		return e.left.Eval(df).(*ArrowSeries).Mul(e.right.Eval(df))
	case opDiv:
		return e.left.Eval(df).(*ArrowSeries).Div(e.right.Eval(df))
	case opMod:
		return e.left.Eval(df).(*ArrowSeries).Mod(e.right.Eval(df))

	case opNeg:
		return e.left.Eval(df).(*ArrowSeries).Neg()
	case opAbs:
		return e.left.Eval(df).(*ArrowSeries).Abs()

	case opEq:
		return e.left.Eval(df).(*ArrowSeries).Eq(e.right.Eval(df))
	case opNe:
		return e.left.Eval(df).(*ArrowSeries).Ne(e.right.Eval(df))
	case opLt:
		return e.left.Eval(df).(*ArrowSeries).Lt(e.right.Eval(df))
	case opLe:
		return e.left.Eval(df).(*ArrowSeries).Le(e.right.Eval(df))
	case opGt:
		return e.left.Eval(df).(*ArrowSeries).Gt(e.right.Eval(df))
	case opGe:
		return e.left.Eval(df).(*ArrowSeries).Ge(e.right.Eval(df))

	case opAnd:
		return e.left.Eval(df).(*ArrowSeries).And(e.right.Eval(df))
	case opOr:
		return e.left.Eval(df).(*ArrowSeries).Or(e.right.Eval(df))
	case opNot:
		return e.left.Eval(df).(*ArrowSeries).Not()

	case opIsNull:
		s := e.left.Eval(df).(*ArrowSeries)
		return isNullSeries(s)
	case opNotNull:
		s := e.left.Eval(df).(*ArrowSeries)
		return isNotNullSeries(s)

	case opFillNA:
		s := e.left.Eval(df).(*ArrowSeries)
		return fillNASeries(s, e.val)

	case opApply:
		s := e.left.Eval(df).(*ArrowSeries)
		return s.MapFloat(e.fn)

	case opAgg:
		// Aggregation: reduce to single value
		s := e.left.Eval(df).(*ArrowSeries)
		vals := make([]float64, 0, s.Len())
		for i := 0; i < s.Len(); i++ {
			if s.NotNull(i) {
				vals = append(vals, s.Float(i))
			}
		}
		result := applyAgg(e.aggFn, vals)
		return newConstFloat64Series("_agg", result, 1)

	case opClip:
		s := e.left.Eval(df).(*ArrowSeries)
		bounds := e.val.([2]float64)
		return s.Clip(bounds[0], bounds[1])

	case opCast:
		s := e.left.Eval(df).(*ArrowSeries)
		dt := e.val.(core.DType)
		return s.AsType(dt)

	case opStringContains:
		s := e.left.Eval(df).(*ArrowSeries)
		pattern := e.val.(string)
		return s.Str().Contains(pattern)

	case opStringReplace:
		s := e.left.Eval(df).(*ArrowSeries)
		pair := e.val.([2]string)
		return s.Str().Replace(pair[0], pair[1])

	case opStringUpper:
		return e.left.Eval(df).(*ArrowSeries).Str().Upper()

	case opStringLower:
		return e.left.Eval(df).(*ArrowSeries).Str().Lower()

	default:
		// Unknown op: return a null series (defensive)
		rows, _ := df.Shape()
		return newConstFloat64Series("_unknown", 0, rows)
	}
}

// --- Helper constructors ---

func newConstFloat64Series(name string, val float64, n int) *ArrowSeries {
	alloc := memory.NewGoAllocator()
	bldr := array.NewFloat64Builder(alloc)
	bldr.Resize(n)
	for i := 0; i < n; i++ {
		bldr.Append(val)
	}
	return NewArrowSeries(name, bldr.NewArray(), nil)
}

func newConstStringSeries(name string, val string, n int) *ArrowSeries {
	alloc := memory.NewGoAllocator()
	bldr := array.NewStringBuilder(alloc)
	bldr.Resize(n)
	for i := 0; i < n; i++ {
		bldr.Append(val)
	}
	return NewArrowSeries(name, bldr.NewArray(), nil)
}

func newConstBoolSeries(name string, val bool, n int) *ArrowSeries {
	alloc := memory.NewGoAllocator()
	bldr := array.NewBooleanBuilder(alloc)
	bldr.Resize(n)
	for i := 0; i < n; i++ {
		bldr.Append(val)
	}
	return NewArrowSeries(name, bldr.NewArray(), nil)
}

func isNullSeries(s *ArrowSeries) *ArrowSeries {
	alloc := memory.NewGoAllocator()
	bldr := array.NewBooleanBuilder(alloc)
	bldr.Resize(s.Len())
	for i := 0; i < s.Len(); i++ {
		bldr.Append(s.IsNull(i))
	}
	return NewArrowSeries(s.Name()+"_isnull", bldr.NewArray(), s.Index())
}

func isNotNullSeries(s *ArrowSeries) *ArrowSeries {
	alloc := memory.NewGoAllocator()
	bldr := array.NewBooleanBuilder(alloc)
	bldr.Resize(s.Len())
	for i := 0; i < s.Len(); i++ {
		bldr.Append(s.NotNull(i))
	}
	return NewArrowSeries(s.Name()+"_notnull", bldr.NewArray(), s.Index())
}

func fillNASeries(s *ArrowSeries, val interface{}) *ArrowSeries {
	alloc := memory.NewGoAllocator()
	bldr := array.NewFloat64Builder(alloc)
	bldr.Resize(s.Len())
	fillVal := toFloat64Val(val)
	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			bldr.Append(fillVal)
		} else {
			bldr.Append(s.Float(i))
		}
	}
	return NewArrowSeries(s.Name(), bldr.NewArray(), s.Index())
}
