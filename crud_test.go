package dbm

import (
	"testing"
)

var tests = []struct {
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
	for _, test := range tests {
		r := q(test.driver, test.query)
		if r != test.expected {
			t.Fatalf("expected: %s got: %s driver: %s", test.expected, r, test.driver)
		}
	}
}
