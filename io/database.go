package io

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/apache/arrow/go/v18/arrow/array"
	"github.com/apache/arrow/go/v18/arrow/memory"
	"github.com/lekeeith/godas/backend/arrow"
	"github.com/lekeeith/godas/core"
)

// DBConfig holds database connection configuration.
type DBConfig struct {
	Driver string // "mysql", "postgres", "sqlite3"
	DSN    string // data source name
}

// MySQLDSN builds a MySQL DSN.
func MySQLDSN(user, password, host string, port int, database string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", user, password, host, port, database)
}

// PostgresDSN builds a PostgreSQL DSN.
func PostgresDSN(user, password, host string, port int, database string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s", user, password, host, port, database)
}

// ReadSQL executes a SQL query and returns a DataFrame.
// Caller must import the driver:
//
//	import _ "github.com/go-sql-driver/mysql"
//	import _ "github.com/jackc/pgx/v5/stdlib"
func ReadSQL(driver, dsn, query string) (*arrow.ArrowDataFrame, error) {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	defer db.Close()
	return ReadSQLFromDB(db, query)
}

// ReadSQLFromDB executes a SQL query on an existing *sql.DB.
func ReadSQLFromDB(db *sql.DB, query string) (*arrow.ArrowDataFrame, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("columns: %w", err)
	}
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("column types: %w", err)
	}

	rawRows := make([][]interface{}, 0)
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		rawRows = append(rawRows, vals)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return buildDFFromRows(cols, colTypes, rawRows)
}

func buildDFFromRows(cols []string, colTypes []*sql.ColumnType, rawRows [][]interface{}) (*arrow.ArrowDataFrame, error) {
	alloc := memory.NewGoAllocator()
	series := make([]*arrow.ArrowSeries, len(cols))

	for j, colName := range cols {
		dbType := ""
		if j < len(colTypes) {
			dbType = colTypes[j].DatabaseTypeName()
		}
		dt := inferSQLColType(dbType, rawRows, j)

		switch dt {
		case core.INT64:
			bldr := array.NewInt64Builder(alloc)
			bldr.Resize(len(rawRows))
			for _, row := range rawRows {
				if row[j] == nil {
					bldr.AppendNull()
				} else {
					bldr.Append(sqlToInt64(row[j]))
				}
			}
			series[j] = arrow.NewArrowSeries(colName, bldr.NewArray(), nil)
			bldr.Release()
		case core.FLOAT64:
			bldr := array.NewFloat64Builder(alloc)
			bldr.Resize(len(rawRows))
			for _, row := range rawRows {
				if row[j] == nil {
					bldr.AppendNull()
				} else {
					bldr.Append(sqlToFloat64(row[j]))
				}
			}
			series[j] = arrow.NewArrowSeries(colName, bldr.NewArray(), nil)
			bldr.Release()
		case core.BOOL:
			bldr := array.NewBooleanBuilder(alloc)
			bldr.Resize(len(rawRows))
			for _, row := range rawRows {
				if row[j] == nil {
					bldr.AppendNull()
				} else {
					bldr.Append(sqlToBool(row[j]))
				}
			}
			series[j] = arrow.NewArrowSeries(colName, bldr.NewArray(), nil)
			bldr.Release()
		default:
			bldr := array.NewStringBuilder(alloc)
			bldr.Resize(len(rawRows))
			for _, row := range rawRows {
				if row[j] == nil {
					bldr.AppendNull()
				} else {
					bldr.Append(fmt.Sprintf("%v", row[j]))
				}
			}
			series[j] = arrow.NewArrowSeries(colName, bldr.NewArray(), nil)
			bldr.Release()
		}
	}
	return arrow.NewDataFrame(series...), nil
}

func inferSQLColType(dbType string, rows [][]interface{}, col int) core.DType {
	switch dbType {
	case "INT", "INTEGER", "BIGINT", "SMALLINT", "TINYINT", "MEDIUMINT",
		"INT2", "INT4", "INT8", "SERIAL", "BIGSERIAL":
		return core.INT64
	case "FLOAT", "DOUBLE", "DECIMAL", "NUMERIC", "REAL", "DOUBLE PRECISION":
		return core.FLOAT64
	case "BOOL", "BOOLEAN":
		return core.BOOL
	}
	boolN, intN, floatN, total := 0, 0, 0, 0
	sample := len(rows)
	if sample > 100 {
		sample = 100
	}
	for i := 0; i < sample; i++ {
		v := rows[i][col]
		if v == nil {
			continue
		}
		total++
		switch v.(type) {
		case bool:
			boolN++
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			intN++
		case float32, float64:
			floatN++
		}
	}
	if total == 0 {
		return core.STRING
	}
	th := float64(total) * 0.8
	if float64(boolN) >= th {
		return core.BOOL
	}
	if float64(intN) >= th {
		return core.INT64
	}
	if float64(intN+floatN) >= th {
		return core.FLOAT64
	}
	return core.STRING
}

func sqlToInt64(v interface{}) int64 {
	switch val := v.(type) {
	case int:
		return int64(val)
	case int8:
		return int64(val)
	case int16:
		return int64(val)
	case int32:
		return int64(val)
	case int64:
		return val
	case uint8:
		return int64(val)
	case uint16:
		return int64(val)
	case uint32:
		return int64(val)
	case uint64:
		return int64(val)
	case float32:
		return int64(val)
	case float64:
		return int64(val)
	}
	return 0
}

func sqlToFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float32:
		return float64(val)
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	}
	return 0
}

func sqlToBool(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case int:
		return val != 0
	case int64:
		return val != 0
	case []byte:
		return string(val) == "true" || string(val) == "1"
	}
	return false
}

// WriteSQL writes a DataFrame to a database table.
func WriteSQL(driver, dsn, tableName string, df *arrow.ArrowDataFrame) error {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()
	return WriteSQLToDB(db, driver, tableName, df)
}

// WriteSQLToDB writes a DataFrame to a database table using an existing connection.
func WriteSQLToDB(db *sql.DB, driver, tableName string, df *arrow.ArrowDataFrame) error {
	cols := df.Columns()
	createSQL := buildCreateTableSQL(tableName, df)
	if _, err := db.Exec(createSQL); err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	ph := "?"
	if driver != "mysql" {
		ph = "$1" // simplified; real impl would iterate
	}
	insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName, strings.Join(cols, ", "), buildPlaceholders(len(cols), driver))

	rows, _ := df.Shape()
	for i := 0; i < rows; i++ {
		vals := make([]interface{}, len(cols))
		for j, name := range cols {
			s := df.Col(name)
			if s.IsNull(i) {
				vals[j] = nil
			} else {
				switch s.Dtype() {
				case core.BOOL:
					vals[j] = s.Bool(i)
				case core.FLOAT32, core.FLOAT64:
					vals[j] = s.Float(i)
				case core.STRING:
					vals[j] = s.String(i)
				default:
					vals[j] = s.Int(i)
				}
			}
		}
		if _, err := db.Exec(insertSQL, vals...); err != nil {
			return fmt.Errorf("insert row %d: %w", i, err)
		}
	}
	_ = ph
	return nil
}

func buildPlaceholders(n int, driver string) string {
	parts := make([]string, n)
	for i := range parts {
		if driver == "mysql" {
			parts[i] = "?"
		} else {
			parts[i] = fmt.Sprintf("$%d", i+1)
		}
	}
	return strings.Join(parts, ", ")
}

func buildCreateTableSQL(tableName string, df *arrow.ArrowDataFrame) string {
	cols := df.Columns()
	defs := make([]string, len(cols))
	for i, name := range cols {
		sqlType := "TEXT"
		switch df.Col(name).Dtype() {
		case core.BOOL:
			sqlType = "BOOLEAN"
		case core.INT8, core.INT16, core.INT32, core.INT64:
			sqlType = "BIGINT"
		case core.FLOAT32, core.FLOAT64:
			sqlType = "DOUBLE PRECISION"
		}
		defs[i] = fmt.Sprintf("%s %s", name, sqlType)
	}
	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", tableName, strings.Join(defs, ", "))
}
