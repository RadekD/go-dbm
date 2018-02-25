package dbm_test

import (
	"context"
	"database/sql"
	"log"
	"reflect"
	"sync/atomic"
	"testing"

	_ "github.com/go-sql-driver/mysql"

	dbm "github.com/RadekD/go-dbm"
)

func setup(t *testing.T) dbm.DB {
	dbPool, err := sql.Open("mysql", "root@tcp(127.0.0.1:3306)/test?collation=utf8mb4_unicode_ci&parseTime=true")
	if err != nil {
		t.Fatal(err)
	}

	db := &dbm.MySQL{
		DB: dbPool,
	}

	return db
}

type JSONTest struct {
	JSON struct {
		A string
	} `db:",json"`
}

func equalTo(expected interface{}) func(t *testing.T, value interface{}) {
	return func(t *testing.T, value interface{}) {
		if expected != value {
			t.Fatalf("got: %+v; expected: %+v", value, expected)
		}
		log.Printf("expected %T, %v, got: %v", expected, expected, value)
	}
}

func equalSliceTo(expected interface{}) func(t *testing.T, value interface{}) {
	return func(t *testing.T, value interface{}) {
		if !reflect.DeepEqual(expected, value) {
			t.Fatalf("expected %T, got: %v", expected, value)
		}
		log.Printf("expected %T, %v, got: %v", expected, expected, value)
	}
}

var selectTests = []struct {
	holder interface{}
	query  string
	testFn func(t *testing.T, value interface{})
}{
	{new(string), "SELECT 'test'", equalTo("test")},
	{new(int), "SELECT 1", equalTo(1)},
	{new(bool), "SELECT 1", equalTo(true)},
	{new(struct{ Name string }), "SELECT 'test' as Name", equalTo(struct{ Name string }{Name: "test"})},
	{new(JSONTest), `SELECT "{\"A\": \"AAAA\"}" as JSON`, equalTo(JSONTest{JSON: struct{ A string }{A: "AAAA"}})},
	{new([]int), "SELECT 1 UNION ALL SELECT 2 UNION ALL SELECT 3", equalSliceTo([]int{1, 2, 3})},
	{new([]string), `SELECT "a" UNION ALL SELECT "b" UNION ALL SELECT "c"`, equalSliceTo([]string{"a", "b", "c"})},
}

func TestSelect(t *testing.T) {
	db := setup(t)

	for _, test := range selectTests {
		test := test
		t.Run(test.query, func(t *testing.T) {
			err := db.Select(test.holder, test.query)
			if err != nil {
				t.Fatal(err)
			}
			value := reflect.ValueOf(test.holder).Elem().Interface()

			test.testFn(t, value)
		})
	}
}

func TestTransaction(t *testing.T) {
	db := setup(t)

	var afterCommit int64

	ctx, cancel := context.WithCancel(context.Background())

	tx, err := db.BeginContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	go func() {
		tx.IfCommited(func() {
			atomic.AddInt64(&afterCommit, 1)
			cancel()
		})

		err = tx.Commit()
		if err != nil {
			t.Fatal(err)
		}
	}()

	<-ctx.Done()

	if atomic.LoadInt64(&afterCommit) != 1 {
		t.Fatal("after commit failed")
	}
}

func TestInvalidTransaction(t *testing.T) {
	db := setup(t)

	var afterCommit int64

	ctx, cancel := context.WithCancel(context.Background())

	tx, err := db.BeginContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	go func() {
		tx.IfCommited(func() {
			atomic.AddInt64(&afterCommit, 1)
			cancel()
		})

		err = tx.Rollback()
		if err != nil {
			t.Fatal(err)
		}
		cancel()
	}()

	<-ctx.Done()

	if atomic.LoadInt64(&afterCommit) == 1 {
		t.Fatal("after commit failed")
	}
}
