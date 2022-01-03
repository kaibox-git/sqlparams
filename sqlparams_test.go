package sqlparams

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestInline(t *testing.T) {
	// some parameters:
	timeDate := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
	digit := 123
	intPtr := &digit
	var otherName, nullName sql.NullString
	otherName.String = `other test string`
	otherName.Valid = true

	str := `ptr test string`
	var strPtr = &str

	// test cases:
	testCases := []struct {
		name     string
		query    string
		params   []interface{}
		expected string
	}{
		{
			name:     `1-1`,
			query:    `$1`,
			params:   []interface{}{123},
			expected: `123`,
		},
		{
			name:     `1-2`,
			query:    `$1`,
			params:   []interface{}{nullName},
			expected: `NULL`,
		},
		{
			name:     `1-3`,
			query:    `$1`,
			params:   []interface{}{otherName},
			expected: `'other test string'`,
		},
		{
			name:     `1-4`,
			query:    `$1, $2`,
			params:   []interface{}{123, `test string`},
			expected: `123, 'test string'`,
		},
		{
			name:     `1-5`,
			query:    `$1, $2, $3`,
			params:   []interface{}{123, `test string`, timeDate},
			expected: `123, 'test string', '2009-11-17 20:34:58.651'`,
		},
		{
			name:     `1-6`,
			query:    `?, ?, ?`,
			params:   []interface{}{123, `test string`, timeDate},
			expected: `123, 'test string', '2009-11-17 20:34:58.651'`,
		},
		{
			name:     `1-7`,
			query:    `$1`,
			params:   []interface{}{timeDate},
			expected: `'2009-11-17 20:34:58.651'`,
		},
		{
			name:     `1-8`,
			query:    `SELECT 1 FROM table`,
			params:   []interface{}{timeDate},
			expected: `placeholder is undefined: SELECT 1 FROM table`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			sql := Inline(testCase.query, testCase.params...)
			require.Equal(t, testCase.expected, sql)
		})
	}

	// other test cases:
	testCases2 := []struct {
		name     string
		query    string
		params   interface{}
		expected string
	}{
		{
			name:     `2-1`,
			query:    `$1`,
			params:   123,
			expected: `123`,
		},
		{
			name:     `2-2`,
			query:    `$1`,
			params:   timeDate,
			expected: `'2009-11-17 20:34:58.651'`,
		},
		{
			name:     `2-3`,
			query:    `$1`,
			params:   otherName,
			expected: `'other test string'`,
		},
		{
			name:     `2-4`,
			query:    `$1`,
			params:   nullName,
			expected: `NULL`,
		},
		{
			name:  `2-5`,
			query: `(:name, :date, :digit)`,
			params: map[string]interface{}{
				`name`:  `test string`,
				`date`:  timeDate,
				`digit`: 123,
			},
			expected: `('test string', '2009-11-17 20:34:58.651', 123)`,
		},
		{
			name:  `2-6`,
			query: `=:name, = :date, =  :digit`,
			params: map[string]interface{}{
				`name`:  `test string`,
				`date`:  timeDate,
				`digit`: 123,
			},
			expected: `='test string', = '2009-11-17 20:34:58.651', =  123`,
		},
		{
			name:  `2-7`,
			query: `select some_field::text WHERE (:name, :date, :digit)`,
			params: map[string]interface{}{
				`name`:  `test string`,
				`date`:  timeDate,
				`digit`: 123,
			},
			expected: `select some_field::text WHERE ('test string', '2009-11-17 20:34:58.651', 123)`,
		},
		{
			name:  `2-8`,
			query: `=:name, =:date, =:digit`,
			params: &map[string]interface{}{ // for a special case...
				`name`:  `test string`,
				`date`:  timeDate,
				`digit`: 123,
			},
			expected: `='test string', ='2009-11-17 20:34:58.651', =123`,
		},
		{
			name:  `2-9`,
			query: `(:name, :date, :digit, :other_name)`,
			params: struct {
				Name      string
				Date      time.Time
				Digit     *int
				OtherName string `db:"other_name"`
			}{
				`test string`,
				timeDate,
				intPtr,
				`other test string`,
			},
			expected: `('test string', '2009-11-17 20:34:58.651', 123, 'other test string')`,
		},
		{
			name:  `2-10`,
			query: `$1, $2, $3, $4`,
			params: struct {
				Name      string
				Date      time.Time
				Digit     *int
				OtherName *string `db:"other_name"`
			}{
				`test string`,
				timeDate,
				intPtr,
				strPtr,
			},
			expected: `'test string', '2009-11-17 20:34:58.651', 123, 'ptr test string'`,
		},
		{
			name:  `2-11`,
			query: `$1, $2, $3, $4, $5`,
			params: &struct {
				Name       string
				Date       *time.Time
				Digit      *int
				OtherDigit int            `db:"other_digit"`
				OtherName  sql.NullString `db:"other_name"`
			}{
				`test string`,
				&timeDate,
				intPtr,
				321,
				otherName,
			},
			expected: `'test string', '2009-11-17 20:34:58.651', 123, 321, 'other test string'`,
		},
		{
			name:  `2-12`,
			query: `?, ?, ?, ?, ?`,
			params: struct {
				Name       string
				Date       *time.Time
				Digit      *int
				OtherDigit int            `db:"other_digit"`
				OtherName  sql.NullString `db:"other_name"`
			}{
				`test string`,
				&timeDate,
				intPtr,
				321,
				nullName,
			},
			expected: `'test string', '2009-11-17 20:34:58.651', 123, 321, NULL`,
		},
	}
	for _, testCase := range testCases2 {
		t.Run(testCase.name, func(t *testing.T) {
			sql := Inline(testCase.query, testCase.params)
			require.Equal(t, testCase.expected, sql)
		})
	}

}
