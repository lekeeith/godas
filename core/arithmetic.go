package core

// Arithmetic defines arithmetic operations on Series.
type Arithmetic interface {
	// Add adds two series element-wise or a series and a scalar.
	Add(other Series) Series
	AddScalar(v float64) Series

	// Sub subtracts element-wise.
	Sub(other Series) Series
	SubScalar(v float64) Series

	// Mul multiplies element-wise.
	Mul(other Series) Series
	MulScalar(v float64) Series

	// Div divides element-wise (returns float64).
	Div(other Series) Series
	DivScalar(v float64) Series

	// Mod computes modulo (integer only).
	Mod(other Series) Series

	// Neg negates all values.
	Neg() Series

	// Abs returns absolute values.
	Abs() Series
}

// Comparison defines comparison operations on Series.
type Comparison interface {
	// Eq returns element-wise equality.
	Eq(other Series) Series
	// Ne returns element-wise inequality.
	Ne(other Series) Series
	// Lt returns element-wise less than.
	Lt(other Series) Series
	// Le returns element-wise less than or equal.
	Le(other Series) Series
	// Gt returns element-wise greater than.
	Gt(other Series) Series
	// Ge returns element-wise greater than or equal.
	Ge(other Series) Series

	// EqScalar returns equality with scalar.
	EqScalar(v float64) Series
	// NeScalar returns inequality with scalar.
	NeScalar(v float64) Series
	// LtScalar returns less than scalar.
	LtScalar(v float64) Series
	// LeScalar returns less than or equal to scalar.
	LeScalar(v float64) Series
	// GtScalar returns greater than scalar.
	GtScalar(v float64) Series
	// GeScalar returns greater than or equal to scalar.
	GeScalar(v float64) Series
}

// Logic defines logical operations on boolean Series.
type Logic interface {
	// And returns element-wise logical AND.
	And(other Series) Series
	// Or returns element-wise logical OR.
	Or(other Series) Series
	// Not returns element-wise logical NOT.
	Not() Series
}

// promoteDType returns the wider type when combining two dtypes.
func PromoteDType(a, b DType) DType {
	if a == b {
		return a
	}
	// Float always wins
	if a.IsFloat() || b.IsFloat() {
		if a == FLOAT64 || b == FLOAT64 {
			return FLOAT64
		}
		return FLOAT32
	}
	// Integer promotion
	if a.IsInteger() && b.IsInteger() {
		// Signed wins over unsigned of same size
		if a == INT64 || b == INT64 || a == UINT64 || b == UINT64 {
			return INT64
		}
		if a == INT32 || b == INT32 || a == UINT32 || b == UINT32 {
			return INT32
		}
		if a == INT16 || b == INT16 || a == UINT16 || b == UINT16 {
			return INT16
		}
		return INT8
	}
	// Mixed: promote to float64
	return FLOAT64
}
