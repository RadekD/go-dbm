package dbm

import (
	"context"
	"database/sql"
)

type MySQL struct {
	context  context.Context
	cancel   context.CancelFunc
	commited bool

	tx *sql.Tx
	DB *sql.DB
}

func (db *MySQL) Select(holder interface{}, query string, args ...interface{}) error {
	if db.tx != nil {
		return selectAll(db.context, db.tx, holder, query, args...)
	}
	return selectAll(context.Background(), db.DB, holder, query, args...)
}
func (db *MySQL) Insert(table string, data interface{}) error {
	if db.tx != nil {
		return insertAll(db.context, db.tx, table, data)
	}
	return insertAll(context.Background(), db.DB, table, data)
}
func (db *MySQL) Exec(query string, args ...interface{}) (sql.Result, error) {
	if db.tx != nil {
		return db.tx.ExecContext(db.context, query, args...)
	}
	return db.DB.Exec(query, args...)
}
func (db *MySQL) Query(query string, args ...interface{}) (*sql.Rows, error) {
	if db.tx == nil {
		return db.tx.QueryContext(db.context, query, args...)
	}
	return db.DB.Query(query, args...)
}
func (db *MySQL) QueryRow(query string, args ...interface{}) *sql.Row {
	if db.tx != nil {
		db.tx.QueryRow(query, args...)
	}
	return db.DB.QueryRow(query, args...)
}

func (db *MySQL) Begin() (Tx, error) {
	return db.BeginContext(context.Background())
}

func (db *MySQL) BeginContext(ctx context.Context) (Tx, error) {
	tx, err := db.DB.Begin()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)

	myTx := MySQL{
		tx:      tx,
		context: ctx,
		cancel:  cancel,
	}
	return &myTx, nil
}

func (db *MySQL) Commit() error {
	if db.tx != nil {
		err := db.tx.Commit()
		if err == sql.ErrTxDone {
			return err
		}

		db.commited = err == nil
		db.cancel()
		return err
	}
	panic("commit without transaction")
}
func (db *MySQL) IfCommited(fn func()) {
	go func() {
		<-db.context.Done()
		if db.commited {
			fn()
		}
	}()
}
func (db *MySQL) Rollback() error {
	if db.tx != nil {
		return db.tx.Rollback()
	}
	panic("rollback without transaction")
}
