package dbm

import (
	"context"
	"database/sql"
)

type CRUD struct {
	driver string
	*sql.DB
}

//Open uses sql.Open to create new pool and return CRUD instance
func Open(driverName, dataSourceName string) (*CRUD, error) {
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	return &CRUD{driver: driverName, DB: db}, nil
}

// Select is used for performing `SELECT` query and unpacking the result into `holder`
// Holder can be any type (byte, int, string, bool, struct) and slice of any type
// If holder is not a slice and query returns more than one row ErrTooManyRows will be returned
// If holder is byte, int, string or bool and query returns a more than one column ErrTooManyColumns will be returned
func (c *CRUD) Select(holder interface{}, query string, args ...interface{}) error {
	return c.SelectContext(context.Background(), holder, query, args...)
}

// SelectContext is used for performing `SELECT` query and unpacking the result into `holder`
// Holder can be any type (byte, int, string, bool, struct) and slice of any type
// If holder is not a slice and query returns more than one row ErrTooManyRows will be returned
// If holder is byte, int, string or bool and query returns a more than one column ErrTooManyColumns will be returned
func (c *CRUD) SelectContext(ctx context.Context, holder interface{}, query string, args ...interface{}) error {
	return selectAll(ctx, c, holder, query, args...)
}

// Insert created and performs INSERT INTO ... query
// If data parameter is not a struct ErrNotAStruct will be returned
func (c *CRUD) Insert(table string, data interface{}) (sql.Result, error) {
	return c.InsertContext(context.Background(), table, data)
}

// InsertContext created and performs INSERT INTO ... query with a given context
// If data parameter is not a struct ErrNotAStruct will be returned
func (c *CRUD) InsertContext(ctx context.Context, table string, data interface{}) (sql.Result, error) {
	return insertAll(ctx, c, c.driver, table, data)
}

// Update create and performs UPDATE table SET ... WHERE `pk` = ? query
// If data parameter is not a struct ErrNotAStruct will be returned
// If data do not have field with `db:",pk"` tag ErrInvalidPkField will be returned
func (c *CRUD) Update(table string, data interface{}) (sql.Result, error) {
	return c.UpdateContext(context.Background(), table, data)
}

// UpdateContext create and performs UPDATE table SET ... WHERE `pk` = ? LIMIT 1 query with a given context
// If data parameter is not a struct ErrNotAStruct will be returned
// If data do not have field with `db:",pk"` tag ErrInvalidPkField will be returned
func (c *CRUD) UpdateContext(ctx context.Context, table string, data interface{}) (sql.Result, error) {
	return updateAll(ctx, c, c.driver, table, data)
}

// Delete created and performs DELETE FROM table WHERE ... LIMIT 1
// If data parameter is not a struct ErrNotAStruct will be returned
// If data do not have field with `db:",pk"` tag ErrInvalidPkField will be returned
func (c *CRUD) Delete(table string, data interface{}) (sql.Result, error) {
	return c.DeleteContext(context.Background(), table, data)
}

// DeleteContext created and performs DELETE FROM table WHERE ... LIMIT 1 with a given context
// If data parameter is not a struct ErrNotAStruct will be returned
// If data do not have field with `db:",pk"` tag ErrInvalidPkField will be returned
func (c *CRUD) DeleteContext(ctx context.Context, table string, data interface{}) (sql.Result, error) {
	return deleteAll(ctx, c, c.driver, table, data)
}

//CRUDTx hold transation creates by crud.Begin / crud.BeingTx
type CRUDTx struct {
	CRUD
	*sql.Tx

	driver string
}

//Begin creates new transaction wrapped with CRUDTx struct
func (c *CRUD) Begin() (*CRUDTx, error) {
	return c.BeginTx(context.Background(), nil)
}

//BeginTx creates new transaction with given context and TxOptions wrapped with CRUDTx struct
func (c *CRUD) BeginTx(ctx context.Context, opts *sql.TxOptions) (*CRUDTx, error) {
	tx, err := c.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &CRUDTx{driver: c.driver, Tx: tx}, nil
}
