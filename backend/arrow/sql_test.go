package arrow

import (
	"testing"

)

func setupSQLTest() (*SQLQuery, *ArrowDataFrame) {
	employees := NewDataFrame(
		NewStringSeries("name", []string{"alice", "bob", "charlie", "dave", "eve"}, nil),
		NewStringSeries("dept", []string{"eng", "eng", "sales", "sales", "eng"}, nil),
		NewFloat64Series("salary", []float64{100, 120, 80, 90, 110}, nil),
		NewInt64Series("age", []int64{25, 30, 35, 28, 22}, nil),
	)

	sql := NewSQL()
	sql.Register("employees", employees)
	return sql, employees
}

func TestSQLSelect(t *testing.T) {
	sql, _ := setupSQLTest()
	result, err := sql.Query("SELECT name, salary FROM employees")
	if err != nil {
		t.Fatal(err)
	}
	rows, cols := result.Shape()
	if rows != 5 || cols != 2 {
		t.Fatalf("shape = (%d,%d), want (5,2)", rows, cols)
	}
}

func TestSQLSelectStar(t *testing.T) {
	sql, _ := setupSQLTest()
	result, err := sql.Query("SELECT * FROM employees")
	if err != nil {
		t.Fatal(err)
	}
	_, cols := result.Shape()
	if cols != 4 {
		t.Fatalf("cols = %d, want 4", cols)
	}
}

func TestSQLWhere(t *testing.T) {
	sql, _ := setupSQLTest()
	result, err := sql.Query("SELECT name, salary FROM employees WHERE salary > 100")
	if err != nil {
		t.Fatal(err)
	}
	if result.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", result.Len())
	}
	if result.Col("name").String(0) != "bob" {
		t.Errorf("name[0] = %q", result.Col("name").String(0))
	}
}

func TestSQLWhereString(t *testing.T) {
	sql, _ := setupSQLTest()
	result, err := sql.Query("SELECT name FROM employees WHERE dept = eng")
	if err != nil {
		t.Fatal(err)
	}
	if result.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", result.Len())
	}
}

func TestSQLWhereAnd(t *testing.T) {
	sql, _ := setupSQLTest()
	result, err := sql.Query("SELECT name FROM employees WHERE salary > 90 AND age < 30")
	if err != nil {
		t.Fatal(err)
	}
	if result.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", result.Len())
	}
}

func TestSQLGroupBy(t *testing.T) {
	sql, _ := setupSQLTest()
	result, err := sql.Query("SELECT dept, SUM(salary) FROM employees GROUP BY dept")
	if err != nil {
		t.Fatal(err)
	}
	rows, _ := result.Shape()
	if rows != 2 {
		t.Fatalf("rows = %d, want 2", rows)
	}
}

func TestSQLGroupByCount(t *testing.T) {
	sql, _ := setupSQLTest()
	result, err := sql.Query("SELECT dept, COUNT(salary) FROM employees GROUP BY dept")
	if err != nil {
		t.Fatal(err)
	}
	rows, _ := result.Shape()
	if rows != 2 {
		t.Fatalf("rows = %d, want 2", rows)
	}
}

func TestSQLOrderBy(t *testing.T) {
	sql, _ := setupSQLTest()
	result, err := sql.Query("SELECT name, salary FROM employees ORDER BY salary DESC")
	if err != nil {
		t.Fatal(err)
	}
	if result.Col("name").String(0) != "bob" {
		t.Errorf("name[0] = %q, want bob", result.Col("name").String(0))
	}
}

func TestSQLLimit(t *testing.T) {
	sql, _ := setupSQLTest()
	result, err := sql.Query("SELECT name FROM employees LIMIT 2")
	if err != nil {
		t.Fatal(err)
	}
	if result.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", result.Len())
	}
}

func TestSQLDistinct(t *testing.T) {
	sql, _ := setupSQLTest()
	result, err := sql.Query("SELECT DISTINCT dept FROM employees")
	if err != nil {
		t.Fatal(err)
	}
	if result.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", result.Len())
	}
}

func TestSQLAlias(t *testing.T) {
	sql, _ := setupSQLTest()
	result, err := sql.Query("SELECT name AS employee_name, salary AS pay FROM employees")
	if err != nil {
		t.Fatal(err)
	}
	cols := result.Columns()
	if cols[0] != "employee_name" || cols[1] != "pay" {
		t.Errorf("columns = %v", cols)
	}
}

func TestSQLChained(t *testing.T) {
	sql, _ := setupSQLTest()
	result, err := sql.Query("SELECT name, salary FROM employees WHERE salary >= 100 ORDER BY salary DESC LIMIT 3")
	if err != nil {
		t.Fatal(err)
	}
	if result.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", result.Len())
	}
	if result.Col("name").String(0) != "bob" {
		t.Errorf("name[0] = %q", result.Col("name").String(0))
	}
}

func TestSQLNotFound(t *testing.T) {
	sql := NewSQL()
	_, err := sql.Query("SELECT * FROM nonexistent")
	if err == nil {
		t.Error("expected error for missing table")
	}
}

func TestSQLSemicolon(t *testing.T) {
	sql, _ := setupSQLTest()
	result, err := sql.Query("SELECT name FROM employees;")
	if err != nil {
		t.Fatal(err)
	}
	if result.Len() != 5 {
		t.Fatalf("Len() = %d, want 5", result.Len())
	}
}

func TestSQLWhereLTE(t *testing.T) {
	sql, _ := setupSQLTest()
	result, err := sql.Query("SELECT name FROM employees WHERE age <= 25")
	if err != nil {
		t.Fatal(err)
	}
	if result.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", result.Len())
	}
}

func TestSQLJoin(t *testing.T) {
	employees := NewDataFrame(
		NewStringSeries("name", []string{"alice", "bob", "charlie"}, nil),
		NewInt64Series("dept_id", []int64{1, 2, 1}, nil),
	)
	departments := NewDataFrame(
		NewInt64Series("dept_id", []int64{1, 2, 3}, nil),
		NewStringSeries("dept_name", []string{"eng", "sales", "hr"}, nil),
	)

	sql := NewSQL()
	sql.Register("employees", employees)
	sql.Register("departments", departments)

	result, err := sql.Query("SELECT name, dept_name FROM employees JOIN departments ON employees.dept_id = departments.dept_id")
	if err != nil {
		t.Fatal(err)
	}
	if result.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", result.Len())
	}
}
