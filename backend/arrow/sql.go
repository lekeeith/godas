package arrow

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lekeeith/godas/core"
)

// SQLQuery parses a SQL SELECT statement and executes it against DataFrames.
// Supported syntax:
//
//	SELECT col1, col2, AGG(col3) FROM table
//	WHERE col > val AND col2 = 'str'
//	GROUP BY col1, col2
//	ORDER BY col1 ASC, col2 DESC
//	LIMIT n
//	JOIN table2 ON table.col = table2.col
type SQLQuery struct {
	sql       string
	tables    map[string]*ArrowDataFrame // registered tables
	parsed    *parsedSQL
}

type parsedSQL struct {
	fields    []sqlField
	table     string
	joins     []sqlJoin
	where     *sqlWhere
	groupBy   []string
	orderBy   []sqlOrderBy
	limit     int
	distinct  bool
}

type sqlField struct {
	expr string
	alias string
	isAgg bool
	aggFn string
}

type sqlJoin struct {
	table string
	on    string // "left.col = right.col"
}

type sqlWhere struct {
	left     string
	operator string
	right    string
	logic    string // "AND" or "OR"
	next     *sqlWhere
}

type sqlOrderBy struct {
	col       string
	ascending bool
}

// NewSQL creates a new SQL query engine.
func NewSQL() *SQLQuery {
	return &SQLQuery{tables: make(map[string]*ArrowDataFrame)}
}

// Register registers a DataFrame as a named table.
func (sql *SQLQuery) Register(name string, df *ArrowDataFrame) *SQLQuery {
	sql.tables[name] = df
	return sql
}

// Query executes a SQL query and returns a DataFrame.
func (sql *SQLQuery) Query(query string) (*ArrowDataFrame, error) {
	parsed, err := parseSQL(query)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}
	sql.parsed = parsed

	df, ok := sql.tables[parsed.table]
	if !ok {
		return nil, fmt.Errorf("table %q not found", parsed.table)
	}

	// Handle JOINs first
	for _, j := range parsed.joins {
		rightTable, ok := sql.tables[j.table]
		if !ok {
			return nil, fmt.Errorf("join table %q not found", j.table)
		}
		parts := strings.Split(j.on, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid JOIN ON: %s", j.on)
		}
		leftCol := strings.TrimSpace(parts[0])
		rightCol := strings.TrimSpace(parts[1])
		// Extract column names (strip table prefix)
		leftCol = stripTablePrefix(leftCol)
		rightCol = stripTablePrefix(rightCol)
		df = df.MergeOn(rightTable, []string{leftCol}, core.Inner).(*ArrowDataFrame)
	}

	// WHERE
	if parsed.where != nil {
		mask := evaluateWhere(df, parsed.where)
		df = df.Filter(mask).(*ArrowDataFrame)
	}

	// GROUP BY + aggregation
	if len(parsed.groupBy) > 0 {
		aggs := make(map[string]core.AggFunc)
		for _, f := range parsed.fields {
			if f.isAgg {
				aggs[f.expr] = parseAggFunc(f.aggFn)
			}
		}
		if len(aggs) > 0 {
			df = df.Agg(parsed.groupBy, aggs).(*ArrowDataFrame)
		}
	}

	// SELECT fields
	if !hasAgg(parsed.fields) || len(parsed.groupBy) > 0 {
		selected := selectFields(df, parsed.fields)
		if selected != nil {
			df = selected
		}
	}

	// ORDER BY
	if len(parsed.orderBy) > 0 {
		cols := make([]string, len(parsed.orderBy))
		asc := make([]bool, len(parsed.orderBy))
		for i, o := range parsed.orderBy {
			cols[i] = o.col
			asc[i] = o.ascending
		}
		df = df.SortBy(cols, asc).(*ArrowDataFrame)
	}

	// DISTINCT
	if parsed.distinct {
		df = df.DropDuplicates(df.Columns(), "first").(*ArrowDataFrame)
	}

	// LIMIT
	if parsed.limit > 0 && df.Len() > parsed.limit {
		df = df.Slice(0, parsed.limit).(*ArrowDataFrame)
	}

	return df, nil
}

func parseSQL(sql string) (*parsedSQL, error) {
	sql = strings.TrimSpace(sql)
	sql = strings.TrimSuffix(sql, ";")

	p := &parsedSQL{}

	// Extract SELECT
	selectIdx := strings.Index(strings.ToUpper(sql), "SELECT ")
	if selectIdx < 0 {
		return nil, fmt.Errorf("missing SELECT")
	}
	sql = sql[selectIdx+7:]

	// Extract DISTINCT
	if strings.HasPrefix(strings.ToUpper(sql), "DISTINCT ") {
		p.distinct = true
		sql = sql[9:]
	}

	// Extract fields (up to FROM)
	fromIdx := strings.Index(strings.ToUpper(sql), " FROM ")
	if fromIdx < 0 {
		return nil, fmt.Errorf("missing FROM")
	}
	fieldsStr := strings.TrimSpace(sql[:fromIdx])
	sql = sql[fromIdx+6:]

	p.fields = parseFields(fieldsStr)

	// Extract table name (up to WHERE/JOIN/GROUP/ORDER/LIMIT/end)
	rest := sql
	tableEnd := len(rest)
	for _, kw := range []string{" WHERE ", " JOIN ", " GROUP BY ", " ORDER BY ", " LIMIT "} {
		idx := strings.Index(strings.ToUpper(rest), kw)
		if idx >= 0 && idx < tableEnd {
			tableEnd = idx
		}
	}
	p.table = strings.TrimSpace(rest[:tableEnd])
	rest = rest[tableEnd:]

	// Parse JOIN
	restUpper := strings.ToUpper(rest)
	for strings.Contains(restUpper, " JOIN ") {
		joinIdx := strings.Index(restUpper, " JOIN ")
		onIdx := strings.Index(restUpper, " ON ")
		if onIdx < 0 {
			break
		}
		joinTable := strings.TrimSpace(rest[joinIdx+6 : onIdx])
		afterOn := rest[onIdx+4:]
		// ON clause ends at next keyword or end
		onEnd := len(afterOn)
		for _, kw := range []string{" WHERE ", " GROUP ", " ORDER ", " LIMIT ", " JOIN "} {
			idx := strings.Index(strings.ToUpper(afterOn), kw)
			if idx >= 0 && idx < onEnd {
				onEnd = idx
			}
		}
		onClause := strings.TrimSpace(afterOn[:onEnd])
		p.joins = append(p.joins, sqlJoin{table: joinTable, on: onClause})
		rest = afterOn[onEnd:]
		restUpper = strings.ToUpper(rest)
	}

	// Parse WHERE
	if idx := strings.Index(restUpper, " WHERE "); idx >= 0 {
		whereStr := rest[idx+7:]
		whereEnd := len(whereStr)
		for _, kw := range []string{" GROUP ", " ORDER ", " LIMIT "} {
			i := strings.Index(strings.ToUpper(whereStr), kw)
			if i >= 0 && i < whereEnd {
				whereEnd = i
			}
		}
		w, err := parseWhere(strings.TrimSpace(whereStr[:whereEnd]))
		if err != nil {
			return nil, err
		}
		p.where = w
		rest = whereStr[whereEnd:]
		restUpper = strings.ToUpper(rest)
	}

	// Parse GROUP BY
	if idx := strings.Index(restUpper, " GROUP BY "); idx >= 0 {
		groupStr := rest[idx+10:]
		groupEnd := len(groupStr)
		for _, kw := range []string{" ORDER ", " LIMIT "} {
			i := strings.Index(strings.ToUpper(groupStr), kw)
			if i >= 0 && i < groupEnd {
				groupEnd = i
			}
		}
		p.groupBy = splitTrim(groupStr[:groupEnd], ",")
		rest = groupStr[groupEnd:]
		restUpper = strings.ToUpper(rest)
	}

	// Parse ORDER BY
	if idx := strings.Index(restUpper, " ORDER BY "); idx >= 0 {
		orderStr := rest[idx+10:]
		orderEnd := len(orderStr)
		if i := strings.Index(strings.ToUpper(orderStr), " LIMIT "); i >= 0 && i < orderEnd {
			orderEnd = i
		}
		p.orderBy = parseOrderBy(strings.TrimSpace(orderStr[:orderEnd]))
		rest = orderStr[orderEnd:]
		restUpper = strings.ToUpper(rest)
	}

	// Parse LIMIT
	if idx := strings.Index(restUpper, " LIMIT "); idx >= 0 {
		limStr := strings.TrimSpace(rest[idx+7:])
		n, err := strconv.Atoi(strings.TrimSpace(limStr))
		if err == nil {
			p.limit = n
		}
	}

	return p, nil
}

func parseFields(s string) []sqlField {
	parts := splitTrim(s, ",")
	fields := make([]sqlField, len(parts))
	for i, p := range parts {
		p = strings.TrimSpace(p)
		// Check for alias (AS) - case insensitive
		upper := strings.ToUpper(p)
		asIdx := strings.Index(upper, " AS ")
		if asIdx >= 0 {
			fields[i].expr = strings.TrimSpace(p[:asIdx])
			fields[i].alias = strings.TrimSpace(p[asIdx+4:])
		} else {
			fields[i].expr = p
			fields[i].alias = p
		}
		// Check for aggregation
		upper = strings.ToUpper(fields[i].expr)
		for _, fn := range []string{"COUNT", "SUM", "AVG", "MIN", "MAX", "STD", "MEDIAN"} {
			if strings.HasPrefix(upper, fn+"(") {
				fields[i].isAgg = true
				fields[i].aggFn = fn
				fields[i].expr = strings.TrimSuffix(fields[i].expr[len(fn)+1:], ")")
				break
			}
		}
	}
	return fields
}

func parseWhere(s string) (*sqlWhere, error) {
	// Split by AND/OR
	for _, logic := range []string{" AND ", " OR "} {
		idx := strings.Index(strings.ToUpper(s), logic)
		if idx >= 0 {
			left, err := parseWhere(strings.TrimSpace(s[:idx]))
			if err != nil {
				return nil, err
			}
			right, err := parseWhere(strings.TrimSpace(s[idx+len(logic):]))
			if err != nil {
				return nil, err
			}
			left.logic = strings.TrimSpace(logic)
			left.next = right
			return left, nil
		}
	}

	// Parse single condition: col op val
	for _, op := range []string{"!=", ">=", "<=", "=", ">", "<", " LIKE ", " NOT LIKE "} {
		idx := strings.Index(strings.ToUpper(s), strings.TrimSpace(op))
		if idx >= 0 {
			left := strings.TrimSpace(s[:idx])
			right := strings.TrimSpace(s[idx+len(op):])
			right = strings.Trim(right, "'\"")
			return &sqlWhere{
				left:     left,
				operator: strings.TrimSpace(op),
				right:    right,
			}, nil
		}
	}
	return nil, fmt.Errorf("cannot parse WHERE: %s", s)
}

func parseOrderBy(s string) []sqlOrderBy {
	parts := splitTrim(s, ",")
	result := make([]sqlOrderBy, len(parts))
	for i, p := range parts {
		p = strings.TrimSpace(p)
		upper := strings.ToUpper(p)
		if strings.HasSuffix(upper, " DESC") {
			result[i].col = strings.TrimSpace(p[:len(p)-5])
			result[i].ascending = false
		} else if strings.HasSuffix(upper, " ASC") {
			result[i].col = strings.TrimSpace(p[:len(p)-4])
			result[i].ascending = true
		} else {
			result[i].col = p
			result[i].ascending = true
		}
	}
	return result
}

func parseAggFunc(name string) core.AggFunc {
	switch strings.ToUpper(name) {
	case "SUM":
		return core.AggSum
	case "AVG":
		return core.AggMean
	case "MIN":
		return core.AggMin
	case "MAX":
		return core.AggMax
	case "COUNT":
		return core.AggCount
	case "STD":
		return core.AggStd
	case "MEDIAN":
		return core.AggMedian
	default:
		return core.AggCount
	}
}

func evaluateWhere(df *ArrowDataFrame, w *sqlWhere) []bool {
	rows, _ := df.Shape()
	mask := make([]bool, rows)

	for i := 0; i < rows; i++ {
		match := evaluateCondition(df, i, w)
		if w.logic == "OR" && w.next != nil {
			match = match || evaluateCondition(df, i, w.next)
		} else if w.next != nil {
			match = match && evaluateCondition(df, i, w.next)
		}
		mask[i] = match
	}
	return mask
}

func evaluateCondition(df *ArrowDataFrame, row int, w *sqlWhere) bool {
	colName := stripTablePrefix(w.left)
	s := df.Col(colName)
	if s.IsNull(row) {
		return false
	}

	val := w.right
	switch s.Dtype() {
	case core.STRING:
		sv := s.String(row)
		switch w.operator {
		case "=":
			return sv == val
		case "!=":
			return sv != val
		case ">", "<", ">=", "<=":
			return compareStrings(sv, val, w.operator)
		case "LIKE":
			return sqlLike(sv, val)
		case "NOT LIKE":
			return !sqlLike(sv, val)
		}
	default:
		fv, _ := strconv.ParseFloat(val, 64)
		sv := s.Float(row)
		switch w.operator {
		case "=":
			return sv == fv
		case "!=":
			return sv != fv
		case ">":
			return sv > fv
		case "<":
			return sv < fv
		case ">=":
			return sv >= fv
		case "<=":
			return sv <= fv
		}
	}
	return false
}

func compareStrings(a, b, op string) bool {
	switch op {
	case ">":
		return a > b
	case "<":
		return a < b
	case ">=":
		return a >= b
	case "<=":
		return a <= b
	}
	return false
}

func sqlLike(s, pattern string) bool {
	// Simple LIKE: % = any, _ = single
	if pattern == "%" {
		return true
	}
	if strings.HasPrefix(pattern, "%") && strings.HasSuffix(pattern, "%") {
		return strings.Contains(s, strings.Trim(pattern, "%"))
	}
	if strings.HasPrefix(pattern, "%") {
		return strings.HasSuffix(s, strings.TrimPrefix(pattern, "%"))
	}
	if strings.HasSuffix(pattern, "%") {
		return strings.HasPrefix(s, strings.TrimSuffix(pattern, "%"))
	}
	return s == pattern
}

func selectFields(df *ArrowDataFrame, fields []sqlField) *ArrowDataFrame {
	if len(fields) == 1 && fields[0].expr == "*" {
		return nil
	}
	series := make([]*ArrowSeries, len(fields))
	for i, f := range fields {
		if f.isAgg {
			continue // already handled in GROUP BY
		}
		s := df.Col(f.expr).(*ArrowSeries)
		if f.alias != f.expr {
			s = s.SetName(f.alias).(*ArrowSeries)
		}
		series[i] = s
	}
	// Filter nil entries (from agg fields)
	valid := make([]*ArrowSeries, 0, len(series))
	for _, s := range series {
		if s != nil {
			valid = append(valid, s)
		}
	}
	if len(valid) == 0 {
		return nil
	}
	return NewDataFrame(valid...)
}

func hasAgg(fields []sqlField) bool {
	for _, f := range fields {
		if f.isAgg {
			return true
		}
	}
	return false
}

func stripTablePrefix(col string) string {
	parts := strings.Split(col, ".")
	if len(parts) == 2 {
		return parts[1]
	}
	return col
}

func splitTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
