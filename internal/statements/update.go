// Copyright 2017 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package statements

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/zhaobingss/xorm/convert"
	"github.com/zhaobingss/xorm/dialects"
	"github.com/zhaobingss/xorm/internal/json"
	"github.com/zhaobingss/xorm/internal/utils"
	"github.com/zhaobingss/xorm/schemas"
)

// BuildUpdates auto generating update columnes and values according a struct
func (statement *Statement) BuildUpdates(bean interface{},
	includeVersion, includeUpdated, includeNil,
	includeAutoIncr, update bool) ([]string, []interface{}, error) {
	//engine := statement.Engine
	table := statement.RefTable
	allUseBool := statement.allUseBool
	useAllCols := statement.useAllCols
	mustColumnMap := statement.MustColumnMap
	nullableMap := statement.NullableMap
	columnMap := statement.ColumnMap
	omitColumnMap := statement.OmitColumnMap
	unscoped := statement.unscoped

	var colNames = make([]string, 0)
	var args = make([]interface{}, 0)
	for _, col := range table.Columns() {
		if !includeVersion && col.IsVersion {
			continue
		}
		if col.IsCreated && !columnMap.Contain(col.Name) {
			continue
		}
		if !includeUpdated && col.IsUpdated {
			continue
		}
		if !includeAutoIncr && col.IsAutoIncrement {
			continue
		}
		if col.IsDeleted && !unscoped {
			continue
		}
		if omitColumnMap.Contain(col.Name) {
			continue
		}
		if len(columnMap) > 0 && !columnMap.Contain(col.Name) {
			continue
		}

		if col.MapType == schemas.ONLYFROMDB {
			continue
		}

		if statement.IncrColumns.IsColExist(col.Name) {
			continue
		} else if statement.DecrColumns.IsColExist(col.Name) {
			continue
		} else if statement.ExprColumns.IsColExist(col.Name) {
			continue
		}

		fieldValuePtr, err := col.ValueOf(bean)
		if err != nil {
			return nil, nil, err
		}

		fieldValue := *fieldValuePtr
		fieldType := reflect.TypeOf(fieldValue.Interface())
		if fieldType == nil {
			continue
		}

		requiredField := useAllCols
		includeNil := useAllCols

		if b, ok := getFlagForColumn(mustColumnMap, col); ok {
			if b {
				requiredField = true
			} else {
				continue
			}
		}

		// !evalphobia! set fieldValue as nil when column is nullable and zero-value
		if b, ok := getFlagForColumn(nullableMap, col); ok {
			if b && col.Nullable && utils.IsZero(fieldValue.Interface()) {
				var nilValue *int
				fieldValue = reflect.ValueOf(nilValue)
				fieldType = reflect.TypeOf(fieldValue.Interface())
				includeNil = true
			}
		}

		var val interface{}

		if fieldValue.CanAddr() {
			if structConvert, ok := fieldValue.Addr().Interface().(convert.Conversion); ok {
				data, err := structConvert.ToDB()
				if err != nil {
					return nil, nil, err
				}

				val = data
				goto APPEND
			}
		}

		if structConvert, ok := fieldValue.Interface().(convert.Conversion); ok {
			data, err := structConvert.ToDB()
			if err != nil {
				return nil, nil, err
			}

			val = data
			goto APPEND
		}

		if fieldType.Kind() == reflect.Ptr {
			if fieldValue.IsNil() {
				if includeNil {
					args = append(args, nil)
					colNames = append(colNames, fmt.Sprintf("%v=?", statement.quote(col.Name)))
				}
				continue
			} else if !fieldValue.IsValid() {
				continue
			} else {
				// dereference ptr type to instance type
				fieldValue = fieldValue.Elem()
				fieldType = reflect.TypeOf(fieldValue.Interface())
				requiredField = true
			}
		}

		switch fieldType.Kind() {
		case reflect.Bool:
			if allUseBool || requiredField {
				val = fieldValue.Interface()
			} else {
				// if a bool in a struct, it will not be as a condition because it default is false,
				// please use Where() instead
				continue
			}
		case reflect.String:
			if !requiredField && fieldValue.String() == "" {
				continue
			}
			// for MyString, should convert to string or panic
			if fieldType.String() != reflect.String.String() {
				val = fieldValue.String()
			} else {
				val = fieldValue.Interface()
			}
		case reflect.Int8, reflect.Int16, reflect.Int, reflect.Int32, reflect.Int64:
			if !requiredField && fieldValue.Int() == 0 {
				continue
			}
			val = fieldValue.Interface()
		case reflect.Float32, reflect.Float64:
			if !requiredField && fieldValue.Float() == 0.0 {
				continue
			}
			val = fieldValue.Interface()
		case reflect.Uint8, reflect.Uint16, reflect.Uint, reflect.Uint32, reflect.Uint64:
			if !requiredField && fieldValue.Uint() == 0 {
				continue
			}
			t := int64(fieldValue.Uint())
			val = reflect.ValueOf(&t).Interface()
		case reflect.Struct:
			if fieldType.ConvertibleTo(schemas.TimeType) {
				t := fieldValue.Convert(schemas.TimeType).Interface().(time.Time)
				if !requiredField && (t.IsZero() || !fieldValue.IsValid()) {
					continue
				}
				val = dialects.FormatColumnTime(statement.dialect, statement.defaultTimeZone, col, t)
			} else if nulType, ok := fieldValue.Interface().(driver.Valuer); ok {
				val, _ = nulType.Value()
				if val == nil && !requiredField {
					continue
				}
			} else {
				if !col.SQLType.IsJson() {
					table, err := statement.tagParser.ParseWithCache(fieldValue)
					if err != nil {
						val = fieldValue.Interface()
					} else {
						if len(table.PrimaryKeys) == 1 {
							pkField := reflect.Indirect(fieldValue).FieldByName(table.PKColumns()[0].FieldName)
							// fix non-int pk issues
							if pkField.IsValid() && (!requiredField && !utils.IsZero(pkField.Interface())) {
								val = pkField.Interface()
							} else {
								continue
							}
						} else {
							return nil, nil, errors.New("Not supported multiple primary keys")
						}
					}
				} else {
					// Blank struct could not be as update data
					if requiredField || !utils.IsStructZero(fieldValue) {
						bytes, err := json.DefaultJSONHandler.Marshal(fieldValue.Interface())
						if err != nil {
							return nil, nil, fmt.Errorf("mashal %v failed", fieldValue.Interface())
						}
						if col.SQLType.IsText() {
							val = string(bytes)
						} else if col.SQLType.IsBlob() {
							val = bytes
						}
					} else {
						continue
					}
				}
			}
		case reflect.Array, reflect.Slice, reflect.Map:
			if !requiredField {
				if fieldValue == reflect.Zero(fieldType) {
					continue
				}
				if fieldType.Kind() == reflect.Array {
					if utils.IsArrayZero(fieldValue) {
						continue
					}
				} else if fieldValue.IsNil() || !fieldValue.IsValid() || fieldValue.Len() == 0 {
					continue
				}
			}

			if col.SQLType.IsText() {
				bytes, err := json.DefaultJSONHandler.Marshal(fieldValue.Interface())
				if err != nil {
					return nil, nil, err
				}
				val = string(bytes)
			} else if col.SQLType.IsBlob() {
				var bytes []byte
				var err error
				if fieldType.Kind() == reflect.Slice &&
					fieldType.Elem().Kind() == reflect.Uint8 {
					if fieldValue.Len() > 0 {
						val = fieldValue.Bytes()
					} else {
						continue
					}
				} else if fieldType.Kind() == reflect.Array &&
					fieldType.Elem().Kind() == reflect.Uint8 {
					val = fieldValue.Slice(0, 0).Interface()
				} else {
					bytes, err = json.DefaultJSONHandler.Marshal(fieldValue.Interface())
					if err != nil {
						return nil, nil, err
					}
					val = bytes
				}
			} else {
				continue
			}
		default:
			val = fieldValue.Interface()
		}

	APPEND:
		args = append(args, val)
		if col.IsPrimaryKey {
			continue
		}
		colNames = append(colNames, fmt.Sprintf("%v = ?", statement.quote(col.Name)))
	}

	return colNames, args, nil
}
