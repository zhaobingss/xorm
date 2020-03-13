package xorm

import (
	"bytes"
	"database/sql"
	"errors"
	"regexp"
	"strings"
)

/// the sql default namespace
var DefaultNamespace = "default_namespace"

/// cache sql template
var sqlMap = map[string]*SqlTemplate{}

/// the regex to be use replace space char in sql
var reg, _ = regexp.Compile("\\s+")

/// the template builder instance
var tplBuilder TemplateBuilder

/// the sql and sql template
type SqlTemplate struct {
	sql string   // sql content
	tpl Template // template for generate the execute sql
}

/// build sql for execute
/// @param key: sql map key, namespace + sql ID
/// @param param: the param to pass to the sql template
func buildSql(key string, param interface{}, session *Session) (string, error) {
	if key == "" {
		return "", errors.New("the map key must be not empty")
	}
	mapper := sqlMap[key]
	if mapper == nil {
		return "", errors.New("can't match the map key: " + key)
	}

	tpl, err := getAndSetTemplate(key, mapper)
	if err != nil {
		return "", err
	}
	bts := &bytes.Buffer{}
	err = tpl.Execute(bts, param)
	val := bts.String()
	val = strings.TrimSpace(val)
	val = reg.ReplaceAllString(val, " ")

	if session.engine.logger.IsShowSQL() {
		session.engine.logger.Infof(val)
	}

	return val, err
}

/// get or set sql template
/// @param key: sql map key, namespace + sql ID
/// @param mapper: SqlTemplate that store the sql map to Template
func getAndSetTemplate(key string, mapper *SqlTemplate) (Template, error) {
	tpl := mapper.tpl
	var err error
	if tpl == nil {
		tpl, err = tplBuilder.New(key, mapper.sql)
		if err != nil {
			return nil, err
		} else {
			mapper.tpl = tpl
		}
	}
	return tpl, nil
}

/// query and fill the result to []map[string]string
/// @param key: sql map key, namespace + sql ID
/// @param param: the param to pass to the sql template
/// @param f: the execute func like eg: db.Query/db.Eexcute
/// @return []map[string]string
/// @return error
func query(key string, param interface{}, session *Session) ([]map[string]string, error) {
	sqlStr, err := buildSql(key, param, session)
	if err != nil {
		return nil, err
	}
	return session.QueryString(sqlStr)
}

/// execute sql
/// @param key: sql map key, namespace + sql ID
/// @param param: the param to pass to the sql template
/// @param f: the execute func like eg: db.Query/db.Eexcute
/// @return sql.Result
/// @return error
func exec(key string, param interface{}, session *Session) (sql.Result, error) {
	sqlStr, err := buildSql(key, param, session)
	if err != nil {
		return nil, err
	}

	result, err := session.exec(sqlStr)
	if err != nil {
		return nil, err
	}
	return result, nil
}

/// query and fill the result to *[]struct or *[]*struct
/// @param dest: the slice struct that the rows will be set eg: *[]struct or *[]*struct
/// @param key: sql map key, namespace + sql ID
/// @param param: the param to pass to the sql template
/// @param f: the execute func like eg: db.Query/db.Eexcute
/// @return error
func selectRows(dest interface{}, key string, param interface{}, session *Session) error {
	sqlStr, err := buildSql(key, param, session)
	if err != nil {
		return err
	}
	return session.SQL(sqlStr).Find(dest)
}

/// fill the result to *struct
/// @param dest: the struct that the rows will be set eg: *struct
/// @param key: sql map key, namespace + sql ID
/// @param param: the param to pass to the sql template
/// @param f: the execute func like eg: db.Query/db.Eexcute
/// @return error
func selectRow(dest interface{}, key string, param interface{}, session *Session) error {
	return selectRows(dest, key, param, session)
}
