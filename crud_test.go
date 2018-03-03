package dbm

import (
	"testing"
)

var testsQ = []struct {
	query    string
	driver   string
	expected string
}{
	{"id = ? AND ee = ?", "mysql", "id = ? AND ee = ?"},
	{"id = ? AND ee = ?", "postgres", "id = $1 AND ee = $2"},
	{"id = ? AND ee = ?", "pgx", "id = $1 AND ee = $2"},
	{"id = ? AND ee = ?", "goracle", "id = :1 AND ee = :2"},
}

func TestQ(t *testing.T) {
	for _, test := range testsQ {
		r := q(test.driver, test.query)
		if r != test.expected {
			t.Fatalf("expected: %s got: %s driver: %s", test.expected, r, test.driver)
		}
	}
}

var testsExpand = []struct {
	query    string
	args     []interface{}
	expected string
}{
	{"id IN(?)", []interface{}{[]string{"a", "b"}}, "id IN(?, ?)"},
	{"id IN(?) AND p = ?", []interface{}{[]string{"a", "b"}, 1}, "id IN(?, ?) AND p = ?"},
	{"id IN(?) AND p = ? AND e IN(?)", []interface{}{[]string{"a", "b", "c"}, 1, []int{1, 2, 3}}, "id IN(?, ?, ?) AND p = ? AND e IN(?, ?, ?)"},
}

func TestExpandQuery(t *testing.T) {
	for _, test := range testsExpand {
		r, _ := expandQuery(test.query, test.args)
		if r != test.expected {
			t.Fatalf("expected: %s got: %s", test.expected, r)
		}
	}
}
