package dbm

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

type execer interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

func insertAll(ctx context.Context, db execer, table string, data interface{}) error {
	baseType := reflect.TypeOf(data)
	baseValue := reflect.ValueOf(data)
	if baseType.Kind() == reflect.Ptr {
		baseType = baseType.Elem()
		baseValue = baseValue.Elem()
		if baseType.Kind() != reflect.Struct {
			return fmt.Errorf("data has to be struct")
		}
	}

	var colNames []string
	var valNames []string
	var values []interface{}

	var pkHolder reflect.Value

	for i := 0; i < baseValue.NumField(); i++ {
		typeField := baseType.Field(i)
		field := baseValue.Field(i)
		if !field.CanInterface() {
			continue
		}

		if tag, ok := typeField.Tag.Lookup("db"); ok {
			if tag == "-" {
				continue
			}
			if strings.Contains(tag, ",pk") {
				pkHolder = field.Addr()
			}
		}

		colNames = append(colNames, typeField.Name)
		valNames = append(valNames, "?")
		values = append(values, field.Interface())
	}

	query := fmt.Sprintf("INSERT INTO %s (`%s`) VALUES (%s)", table, strings.Join(colNames, "`, `"), strings.Join(valNames, ", "))
	result, err := db.ExecContext(ctx, query, values...)
	if err != nil {
		return err
	}

	if pkHolder.IsValid() {
		i, err := result.LastInsertId()
		if err != nil {
			return err
		}
		reflect.Indirect(pkHolder).SetInt(i)
	}
	return nil
}
