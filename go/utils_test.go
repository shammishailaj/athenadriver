// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package athenadriver

import (
	"database/sql"
	"database/sql/driver"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/stretchr/testify/assert"
	"math"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestScanNullString(t *testing.T) {
	s, e := scanNullString(nil)
	assert.Nil(t, e)
	assert.Equal(t, s, sql.NullString{})

	s, e = scanNullString("nil")
	assert.Nil(t, e)
	assert.True(t, s.Valid)

	s, e = scanNullString(1)
	assert.NotNil(t, e)
	assert.False(t, s.Valid)
}

func TestColsToCSV(t *testing.T) {
	sqlRows := sqlmock.NewRows([]string{"one", "two", "three"})
	rows := mockRowsToSQLRows(sqlRows)
	expected := ColsToCSV(rows)
	assert.Equal(t, expected, "one,two,three\n")
	assert.Equal(t, "", ColsToCSV(nil))
}

func TestRowsToCSV(t *testing.T) {
	sqlRows := sqlmock.NewRows([]string{"one", "two", "three"})
	sqlRows.AddRow("1", "2", "3")
	rows := mockRowsToSQLRows(sqlRows)
	expected := RowsToCSV(rows)
	assert.Equal(t, expected, "1,2,3\n")

	s := RowsToCSV(nil)
	assert.Equal(t, "", s)
}

func TestColsRowsToCSV(t *testing.T) {
	sqlRows := sqlmock.NewRows([]string{"one", "two", "three"})
	sqlRows.AddRow("1", "2", "3")
	rows := mockRowsToSQLRows(sqlRows)
	expected := ColsRowsToCSV(rows)
	assert.Equal(t, expected, "one,two,three\n1,2,3\n")
}

func TestIsSelectStatement(t *testing.T) {
	assert.True(t, colInFirstPage("SELECT"))
	assert.True(t, colInFirstPage(" SELECT"))
	assert.True(t, colInFirstPage("select"))
}

func TestIsInsetStatement(t *testing.T) {
	assert.True(t, isInsertStatement("INSERT"))
	assert.True(t, isInsertStatement("     INSERT"))
	assert.True(t, isInsertStatement("insert"))
}

func TestRandInt8(t *testing.T) {
	s := randInt8()
	i, err := strconv.ParseInt(*s, 10, 8)
	assert.True(t, math.MinInt8 <= i && i <= math.MaxInt8)
	assert.Nil(t, err)
}

func TestRandInt16(t *testing.T) {
	s := randInt16()
	i, err := strconv.ParseInt(*s, 10, 16)
	assert.True(t, math.MinInt16 <= i && i <= math.MaxInt16)
	assert.Nil(t, err)
}

func TestRandInt(t *testing.T) {
	s := randInt()
	i, err := strconv.ParseInt(*s, 10, 32)
	assert.True(t, math.MinInt32 <= i && i <= math.MaxInt32)
	assert.Nil(t, err)
}

func TestRandInt64(t *testing.T) {
	s := randUInt64()
	_, err := strconv.ParseUint(*s, 10, 64)
	assert.Nil(t, err)
}

func TestRandFloat32(t *testing.T) {
	s := randFloat32()
	i, err := strconv.ParseFloat(*s, 32)
	assert.True(t, math.SmallestNonzeroFloat32 <= i && i <= math.MaxFloat32)
	assert.Nil(t, err)
}

func TestRandFloat64(t *testing.T) {
	s := randFloat64()
	i, err := strconv.ParseFloat(*s, 64)
	assert.True(t, math.SmallestNonzeroFloat64 <= i && i <= math.MaxFloat64)
	assert.Nil(t, err)
}

func TestRandRow(t *testing.T) {
	c1 := newColumnInfo("c1", nil)
	r := randRow([]*athena.ColumnInfo{c1})
	assert.Equal(t, len(r.Data), 1)
	assert.Equal(t, *r.Data[0].VarCharValue, "a\tb")

	types := []string{"tinyint", "smallint", "integer", "bigint", "float", "real", "double",
		"json", "char", "varchar", "varbinary", "row", "string", "binary",
		"struct", "interval year to month", "interval day to second", "decimal",
		"ipaddress", "array", "map", "unknown", "boolean", "date", "time", "time with time zone",
		"timestamp with time zone", "timestamp", "weird_type"}
	for _, ty := range types {
		c1 := newColumnInfo("c1", ty)
		r := randRow([]*athena.ColumnInfo{c1})
		assert.Equal(t, len(r.Data), 1)
	}
}

func TestNamedValueToValue(t *testing.T) {
	dn := driver.NamedValue{
		Name: "abc",
	}
	d := []driver.NamedValue{
		dn,
	}
	v := namedValueToValue(d)
	assert.Equal(t, len(v), 1)
}

type aType struct {
	S string
}

func TestValueToNamedValue(t *testing.T) {
	dn := aType{
		S: "abc",
	}
	d := []driver.Value{
		dn,
	}
	v := valueToNamedValue(d)
	assert.Equal(t, len(v), 1)
	assert.True(t, v[0].Name == "")
	assert.True(t, v[0].Ordinal == 1)
	assert.True(t, v[0].Value.(aType).S == "abc")
}

func TestIsQueryTimeOut(t *testing.T) {
	assert.False(t, isQueryTimeOut(time.Now(), athena.StatementTypeDdl))
	assert.False(t, isQueryTimeOut(time.Now(), athena.StatementTypeDml))
	assert.False(t, isQueryTimeOut(time.Now(), athena.StatementTypeUtility))
	now := time.Now()
	OneHourAgo := now.Add(-3600 * time.Second)
	assert.True(t, isQueryTimeOut(OneHourAgo, athena.StatementTypeDml))
	assert.False(t, isQueryTimeOut(OneHourAgo, athena.StatementTypeDdl))
	assert.False(t, isQueryTimeOut(OneHourAgo, "UNKNOWN"))
}

func TestEscapeBytesBackslash(t *testing.T) {
	r := escapeBytesBackslash([]byte{}, []byte{'\x00'})
	assert.Equal(t, string(r), "\\0")

	r = escapeBytesBackslash([]byte{}, []byte{'\n'})
	assert.Equal(t, string(r), "\\n")

	r = escapeBytesBackslash([]byte{}, []byte{'\r'})
	assert.Equal(t, string(r), "\\r")

	r = escapeBytesBackslash([]byte{}, []byte{'\x1a'})
	assert.Equal(t, string(r), "\\Z")

	r = escapeBytesBackslash([]byte{}, []byte{'\''})
	assert.Equal(t, string(r), `\'`)

	r = escapeBytesBackslash([]byte{}, []byte{'"'})
	assert.Equal(t, string(r), `\"`)

	r = escapeBytesBackslash([]byte{}, []byte{'\\'})
	assert.Equal(t, string(r), `\\`)

	r = escapeBytesBackslash([]byte{}, []byte{'x'})
	assert.Equal(t, string(r), `x`)
}

func TestGetFromEnvVal(t *testing.T) {
	os.Setenv("henrywu_test", "1")
	assert.Equal(t, GetFromEnvVal([]string{"henrywu_test"}), "1")
	assert.Equal(t, GetFromEnvVal([]string{"wufuheng", "henrywu_test"}), "1")
	os.Unsetenv("henrywu_test")
	assert.Equal(t, GetFromEnvVal([]string{"henrywu_test"}), "")
}

func TestPrintCost(t *testing.T) {
	ping := "SELECTExecContext_OK_QID"
	stat := athena.QueryExecutionStateSucceeded
	o := &athena.GetQueryExecutionOutput{
		QueryExecution: &athena.QueryExecution{
			Query:            &ping,
			QueryExecutionId: &ping,
			Status: &athena.QueryExecutionStatus{
				State: &stat,
			},
			Statistics: &athena.QueryExecutionStatistics{
				DataScannedInBytes: nil,
			},
		},
	}
	printCost(nil)
	printCost(o)
	cost := int64(123)
	o.QueryExecution.Statistics.DataScannedInBytes = &cost
	printCost(o)
	cost = int64(12345678)
	o.QueryExecution.Statistics.DataScannedInBytes = &cost
	printCost(o)
}
