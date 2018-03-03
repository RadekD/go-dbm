package dbm

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//ErrTooManyColumns means that provided query in Select function returned more columns that holder can hold
var ErrTooManyColumns = errors.New("crud: too many columns")

//ErrTooManyRows means that provided query in Select function returned more rows that holder can hold
// usually that means that holder is not a slice or lacking a LIMIT in query
var ErrTooManyRows = errors.New("crud: too many rows")

//ErrInvalidPkField means that provided struct do not have field with `db:",pk"` tag
var ErrInvalidPkField = errors.New("crud: struct with invalid pk field")

//ErrNotAStruct means that provided data is not a struct and insert cannot be performed
var ErrNotAStruct = errors.New("crud: data not a struct")

type execer interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

func insertAll(ctx context.Context, db execer, driver string, table string, data interface{}) (sql.Result, error) {
	names, values, pkName, pkValue, pkHolder, err := getNamesAndValues(data)
	if err != nil {
		return nil, err
	}

	var placeholders []string
	for range names {
		placeholders = append(placeholders, "?")
	}

	names = append(names, pkName)
	values = append(values, pkValue)
	placeholders = append(placeholders, "?")

	query := fmt.Sprintf("INSERT INTO %s (`%s`) VALUES (%s)", table, strings.Join(names, "`, `"), strings.Join(placeholders, ", "))
	result, err := db.ExecContext(ctx, q(driver, query), values...)
	if err != nil {
		return nil, err
	}

	if pkHolder.IsValid() {
		i, err := result.LastInsertId()
		if err != nil {
			return result, err
		}
		reflect.Indirect(pkHolder).SetInt(i)
	}
	return result, nil
}

func updateAll(ctx context.Context, db execer, driver string, table string, data interface{}) (sql.Result, error) {
	names, values, pkName, pkValue, _, err := getNamesAndValues(data)
	if err != nil {
		return nil, err
	}
	if pkName == "" {
		return nil, ErrInvalidPkField
	}

	query := fmt.Sprintf("UPDATE %s SET `%s` = ? WHERE `%s` = ? LIMIT 1", table, strings.Join(names, "` = ? `"), pkName)

	values = append(values, pkValue)
	return db.ExecContext(ctx, q(driver, query), values...)
}

func deleteAll(ctx context.Context, db execer, driver string, table string, data interface{}) (sql.Result, error) {
	names, values, pkName, pkValue, _, err := getNamesAndValues(data)
	if err != nil {
		return nil, err
	}
	if pkName == "" {
		return nil, ErrInvalidPkField
	}
	names = append(names, pkName)
	values = append(values, pkValue)

	query := fmt.Sprintf("DELETE FROM %s WHERE `%s` = ? LIMIT 1", table, strings.Join(names, "` = ? AND `"))
	return db.ExecContext(ctx, q(driver, query), values...)
}

type queryer interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

func selectAll(ctx context.Context, db queryer, holder interface{}, driver string, query string, args ...interface{}) error {
	holderType := reflect.TypeOf(holder)
	if holderType.Kind() != reflect.Ptr {
		return fmt.Errorf("holder must be a pointer")
	}
	holderValue := reflect.ValueOf(holder).Elem()
	baseType := holderType.Elem()

	if driver == "mysql" && len(args) > 0 {
		query, args = expandQuery(query, args)
	}

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

func expandQuery(query string, args []interface{}) (string, []interface{}) {
	var newQuery string
	var newArgs []interface{}

	queryParts := strings.Split(query, "?")
	for i, arg := range args {
		if reflect.TypeOf(arg).Kind() == reflect.Slice {
			len := reflect.ValueOf(arg).Len()

			placeholders := strings.Repeat("?, ", len)[:(len*3)-2]
			newQuery += queryParts[i] + placeholders

			for j := 0; j < len; j++ {
				newArgs = append(newArgs, reflect.ValueOf(args[i]).Index(j).Interface())
			}
		} else {
			newQuery += queryParts[i] + "?"
			newArgs = append(newArgs, args[i])
		}
	}
	newQuery += queryParts[len(args)]

	return newQuery, newArgs
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

func getNamesAndValues(data interface{}) (
	names []string,
	values []interface{},
	pkName string,
	pkValue interface{},
	pkHolder reflect.Value,
	err error,
) {
	baseType := reflect.TypeOf(data)
	baseValue := reflect.ValueOf(data)
	if baseType.Kind() == reflect.Ptr {
		baseType = baseType.Elem()
		baseValue = baseValue.Elem()
	}
	if baseType.Kind() != reflect.Struct {
		err = ErrNotAStruct
		return
	}

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
				pkName = typeField.Name
				pkValue = field.Interface()
				pkHolder = field.Addr()
				continue
			}
		}

		names = append(names, typeField.Name)
		values = append(values, field.Interface())
	}
	return
}

var qrx = regexp.MustCompile(`\?`)

// q converts "?" characters to $1, $2, $n on postgres, :1, :2, :n on Oracle
func q(driver string, sql string) string {
	var pref string
	switch driver {
	case "postgres", "pgx":
		pref = "$"
	case "goracle":
		pref = ":"
	default:
		return sql
	}
	n := 0
	return qrx.ReplaceAllStringFunc(sql, func(string) string {
		n++
		return pref + strconv.Itoa(n)
	})
}
