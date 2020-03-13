package xorm

import (
	"database/sql"
)

/// execute the sql with a can ignore result
/// @param key: sql map key, namespace + sql ID
/// @param param: the param to pass to the sql template
func (session *Session) ExecTpl(key string, data interface{}) (sql.Result, error) {
	return exec(key, data, session)
}

/// execute the sql and set result to []map[string]string
/// @param key: sql map key, namespace + sql ID
/// @param param: the param to pass to the sql template
func (session *Session) QueryTpl(key string, data interface{}) ([]map[string]string, error) {
	return query(key, data, session)
}

/// execute sql and set the result to a slice dest
/// @param the result will be set to dest, and the dest must be like eg: *[]*struct or *[]struct
/// @param key: sql map key, namespace + sql ID
/// @param param: the param to pass to the sql template
func (session *Session) SelectTpl(dest interface{}, key string, param interface{}) error {
	return selectRows(dest, key, param, session)
}

/// execute sql and set the result to a struct dest
/// @param the result will be set to dest, and the dest must be like eg: *struct
/// @param key: sql map key, namespace + sql ID
/// @param param: the param to pass to the sql template
/// @return error: ERR_NOT_GOT_RECORD,ERR_MORE_THAN_ONE_RECORD,...
/// ERR_NOT_GOT_RECORD indicate that not got any recode from the database
/// ERR_MORE_THAN_ONE_RECORD indicate that got more than one record from database
func (session *Session) SelectOneTpl(dest interface{}, key string, param interface{}) error {
	return selectRow(dest, key, param, session)
}
