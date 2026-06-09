package core

// JoinType specifies the type of join operation.
type JoinType int

const (
	Inner JoinType = iota
	Left
	Right
	Outer
	Cross
)

func (j JoinType) String() string {
	switch j {
	case Inner:
		return "inner"
	case Left:
		return "left"
	case Right:
		return "right"
	case Outer:
		return "outer"
	case Cross:
		return "cross"
	default:
		return "unknown"
	}
}

// AggFunc specifies an aggregation function.
type AggFunc int

const (
	AggSum AggFunc = iota
	AggMean
	AggMedian
	AggMin
	AggMax
	AggCount
	AggStd
	AggVar
	AggFirst
	AggLast
	AggNUnique
)

func (a AggFunc) String() string {
	switch a {
	case AggSum:
		return "sum"
	case AggMean:
		return "mean"
	case AggMedian:
		return "median"
	case AggMin:
		return "min"
	case AggMax:
		return "max"
	case AggCount:
		return "count"
	case AggStd:
		return "std"
	case AggVar:
		return "var"
	case AggFirst:
		return "first"
	case AggLast:
		return "last"
	case AggNUnique:
		return "nunique"
	default:
		return "unknown"
	}
}
