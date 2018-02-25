package dbm // import "dejnek.pl/go-dbm"

import (
	"context"
	"database/sql"
	"errors"
)

//ErrTooManyColumns means that provided query in Select function returned more columns that holder can hold
var ErrTooManyColumns = errors.New("DB: too many columns")

//ErrTooManyRows means that provided query in Select function returned more rows that holder can hold
// usually that means that holder is not a slice or lacking a LIMIT in query
var ErrTooManyRows = errors.New("DB: too many rows")

//DB simple interface
type DB interface {
	Select(holder interface{}, query string, args ...interface{}) error
	Insert(table string, data interface{}) error

	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row

	Begin() (Tx, error)
	BeginContext(ctx context.Context) (Tx, error)
}

//Tx transaction
type Tx interface {
	DB

	Commit() error
	Rollback() error

	IfCommited(func())
}
