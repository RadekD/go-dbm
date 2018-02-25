package dbm // import "dejnek.pl/go-dbm"

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

type querier interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

func selectAll(ctx context.Context, db querier, holder interface{}, query string, args ...interface{}) error {
	holderType := reflect.TypeOf(holder)
	if holderType.Kind() != reflect.Ptr {
		return fmt.Errorf("holder must be a pointer")
	}
	holderValue := reflect.ValueOf(holder).Elem()
	baseType := holderType.Elem()

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for r := 0; rows.Next(); r++ {
		switch baseType.Kind() {
		case reflect.Struct:
			if r > 0 {
				return ErrTooManyColumns
			}

			if scanStruct(rows, holderValue, baseType) != nil {
				return err
			}
		case reflect.Slice:
			toAppend := reflect.New(baseType.Elem())

			switch baseType.Elem().Kind() {
			case reflect.Struct:
				if scanStruct(rows, reflect.Indirect(toAppend), baseType.Elem()) != nil {
					return err
				}
			default:
				if rows.Scan(toAppend.Interface()) != nil {
					return err
				}
			}
			holderValue.Set(reflect.Append(holderValue, reflect.Indirect(toAppend)))
		default:
			if r > 0 {
				return ErrTooManyColumns
			}

			if cols, err := rows.Columns(); err != nil || len(cols) > 1 {
				if err != nil {
					return err
				}
				return ErrTooManyColumns
			}

			if rows.Scan(holder) != nil {
				return err
			}
		}
	}
	return rows.Err()
}

func scanStruct(rows *sql.Rows, baseValue reflect.Value, baseType reflect.Type) error {
	if _, ok := baseValue.Addr().Interface().(sql.Scanner); ok {
		err := rows.Scan(baseValue.Addr().Interface())
		if err != nil {
			return err
		}
	}
	if _, ok := baseValue.Addr().Interface().(*time.Time); ok {
		err := rows.Scan(baseValue.Addr().Interface())
		if err != nil {
			return err
		}
	}

	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	var toScan = make([]interface{}, len(cols))
	for i, colName := range cols {
		field, ok := baseType.FieldByNameFunc(func(match string) bool {
			return strings.EqualFold(match, colName)
		})

		if !ok {
			toScan[i] = &dummy{}
			continue
		}
		realTarget := baseValue.FieldByIndex(field.Index).Addr().Interface()
		if tag, ok := field.Tag.Lookup("db"); ok {
			if tag == "-" {
				continue
			}
			if strings.Contains(tag, ",json") {
				target := &jsonScanner{target: realTarget}
				toScan[i] = target
				continue
			}
		}

		toScan[i] = realTarget
	}

	return rows.Scan(toScan...)
}

type jsonScanner struct {
	target interface{}
}

func (j *jsonScanner) Scan(value interface{}) error {
	return json.Unmarshal(value.([]byte), j.target)
}

type dummy struct{}

func (*dummy) Scan(value interface{}) error {
	return nil
}
