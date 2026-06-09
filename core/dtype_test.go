package core

import "testing"

func TestDTypeString(t *testing.T) {
	tests := []struct {
		dt   DType
		want string
	}{
		{BOOL, "bool"},
		{INT8, "int8"},
		{INT16, "int16"},
		{INT32, "int32"},
		{INT64, "int64"},
		{UINT8, "uint8"},
		{UINT16, "uint16"},
		{UINT32, "uint32"},
		{UINT64, "uint64"},
		{FLOAT32, "float32"},
		{FLOAT64, "float64"},
		{STRING, "string"},
		{TIMESTAMP, "timestamp"},
		{DURATION, "duration"},
		{CATEGORY, "category"},
		{DType(999), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.dt.String(); got != tt.want {
			t.Errorf("DType(%d).String() = %q, want %q", tt.dt, got, tt.want)
		}
	}
}

func TestDTypeIsInteger(t *testing.T) {
	intTypes := []DType{INT8, INT16, INT32, INT64, UINT8, UINT16, UINT32, UINT64}
	for _, dt := range intTypes {
		if !dt.IsInteger() {
			t.Errorf("%s.IsInteger() = false, want true", dt)
		}
	}
	nonInts := []DType{BOOL, FLOAT32, FLOAT64, STRING, TIMESTAMP}
	for _, dt := range nonInts {
		if dt.IsInteger() {
			t.Errorf("%s.IsInteger() = true, want false", dt)
		}
	}
}

func TestDTypeIsFloat(t *testing.T) {
	if !FLOAT32.IsFloat() {
		t.Error("FLOAT32.IsFloat() = false")
	}
	if !FLOAT64.IsFloat() {
		t.Error("FLOAT64.IsFloat() = false")
	}
	if INT64.IsFloat() {
		t.Error("INT64.IsFloat() = true")
	}
}

func TestDTypeIsNumeric(t *testing.T) {
	if !INT64.IsNumeric() {
		t.Error("INT64.IsNumeric() = false")
	}
	if !FLOAT64.IsNumeric() {
		t.Error("FLOAT64.IsNumeric() = false")
	}
	if STRING.IsNumeric() {
		t.Error("STRING.IsNumeric() = true")
	}
}

func TestDTypeIsSigned(t *testing.T) {
	for _, dt := range []DType{INT8, INT16, INT32, INT64} {
		if !dt.IsSigned() {
			t.Errorf("%s.IsSigned() = false", dt)
		}
	}
	for _, dt := range []DType{UINT8, UINT16, UINT32, UINT64, FLOAT64} {
		if dt.IsSigned() {
			t.Errorf("%s.IsSigned() = true", dt)
		}
	}
}
