// Package core defines the fundamental types and interfaces for godas.
package core

// DType represents the data type of a column or series.
type DType int

const (
	BOOL       DType = iota // bool
	INT8                      // int8
	INT16                     // int16
	INT32                     // int32
	INT64                     // int64
	UINT8                     // uint8
	UINT16                    // uint16
	UINT32                    // uint32
	UINT64                    // uint64
	FLOAT32                   // float32
	FLOAT64                   // float64
	STRING                    // string
	TIMESTAMP                 // time.Time
	DURATION                  // time.Duration
	CATEGORY                  // categorical
)

// String returns the human-readable name of the DType.
func (d DType) String() string {
	switch d {
	case BOOL:
		return "bool"
	case INT8:
		return "int8"
	case INT16:
		return "int16"
	case INT32:
		return "int32"
	case INT64:
		return "int64"
	case UINT8:
		return "uint8"
	case UINT16:
		return "uint16"
	case UINT32:
		return "uint32"
	case UINT64:
		return "uint64"
	case FLOAT32:
		return "float32"
	case FLOAT64:
		return "float64"
	case STRING:
		return "string"
	case TIMESTAMP:
		return "timestamp"
	case DURATION:
		return "duration"
	case CATEGORY:
		return "category"
	default:
		return "unknown"
	}
}

// IsInteger returns true if the dtype is any integer type.
func (d DType) IsInteger() bool {
	return d >= INT8 && d <= UINT64
}

// IsFloat returns true if the dtype is a floating-point type.
func (d DType) IsFloat() bool {
	return d == FLOAT32 || d == FLOAT64
}

// IsNumeric returns true if the dtype is numeric (integer or float).
func (d DType) IsNumeric() bool {
	return d.IsInteger() || d.IsFloat()
}

// IsSigned returns true if the dtype is a signed integer type.
func (d DType) IsSigned() bool {
	return d >= INT8 && d <= INT64
}
