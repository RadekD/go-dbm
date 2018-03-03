package dbm_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/RadekD/go-dbm"

	_ "github.com/go-sql-driver/mysql"
)

func setup(t *testing.T) *dbm.CRUD {
	crud, err := dbm.Open("mysql", "root@tcp(127.0.0.1:3306)/test?collation=utf8mb4_unicode_ci&parseTime=true")
	if err != nil {
		t.Fatal(err)
	}
	return crud
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
		//log.Printf("expected %T, %v, got: %v", expected, expected, value)
	}
}

func equalSliceTo(expected interface{}) func(t *testing.T, value interface{}) {
	return func(t *testing.T, value interface{}) {
		if !reflect.DeepEqual(expected, value) {
			t.Fatalf("expected %T, got: %v", expected, value)
		}
		//log.Printf("expected %T, %v, got: %v", expected, expected, value)
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

type insert struct {
	ID   int `db:",pk"`
	Name string
}

func TestInsert(t *testing.T) {
	db := setup(t)

	testInsert := insert{Name: "test"}

	result, err := db.Insert("test", &testInsert)
	if err != nil {
		t.Fatal(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	if testInsert.ID != int(id) {
		t.Fatalf("expected %T, got: %v", testInsert.ID, id)
	}

	_, err = db.Insert("test", 1235)
	if err != dbm.ErrNotAStruct {
		t.Fatal(err)
	}
}

func TestUpdate(t *testing.T) {
	db := setup(t)

	var sel insert
	err := db.Select(&sel, "SELECT * FROM test WHERE ID = ?", 12)
	if err != nil {
		t.Fatal(err)
	}

	var randomstring = time.Now().String()

	sel.Name = randomstring

	result, err := db.Update("test", &sel)
	if err != nil {
		t.Fatal(err)
	}
	if r, err := result.RowsAffected(); err != nil || r != 1 {
		t.Fatal(err)
	}

	err = db.Select(&sel, "SELECT * FROM test WHERE ID = ?", 12)
	if err != nil {
		t.Fatal(err)
	}
	if sel.Name != randomstring {
		t.Fatal("wrong update")
	}

}

func TestDelete(t *testing.T) {
	db := setup(t)

	var sel insert
	err := db.Select(&sel, "SELECT * FROM test WHERE ID = ?", 1)
	if err != nil {
		t.Fatal(err)
	}

	result, err := db.Delete("test", &sel)
	if err != nil {
		t.Fatal(err)
	}
	_, err = result.RowsAffected()
	if err != nil {
		t.Fatal(err)
	}
}

func TestArraySelect(t *testing.T) {
	db := setup(t)

	var sel []struct{ ID int }
	db.Select(&sel, "SELECT ID FROM test WHERE ID IN(?)", []int{2, 3, 4})

	if len(sel) != 3 {
		t.Fail()
	}
}
