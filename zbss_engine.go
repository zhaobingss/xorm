package xorm

import (
	"database/sql"
	"errors"
	"github.com/beevik/etree"
	"io/ioutil"
	"os"
	"strings"
)

/// execute the sql with a can ignore result
/// @param key: sql map key, namespace + sql ID
/// @param param: the param to pass to the sql template
func (engine *Engine) ExecTpl(key string, param interface{}) (sql.Result, error) {
	session := engine.NewSession()
	defer session.Close()
	return session.ExecTpl(key, param)
}

/// execute the sql and set result to []map[string]string
/// @param key: sql map key, namespace + sql ID
/// @param param: the param to pass to the sql template
func (engine *Engine) QueryTpl(key string, param interface{}) ([]map[string]string, error) {
	session := engine.NewSession()
	defer session.Close()
	return session.QueryTpl(key, param)
}

/// execute sql and set the result to a slice dest
/// @param the result will be set to dest, and the dest must be like eg: *[]*struct or *[]struct
/// @param key: sql map key, namespace + sql ID
/// @param param: the param to pass to the sql template
func (engine *Engine) SelectTpl(dest interface{}, key string, param interface{}) error {
	session := engine.NewSession()
	defer session.Close()
	return session.SelectTpl(dest, key, param)
}

/// execute sql and set the result to a struct dest
/// @param the result will be set to dest, and the dest must be like eg: *struct
/// @param key: sql map key, namespace + sql ID
/// @param param: the param to pass to the sql template
/// @return error: ERR_NOT_GOT_RECORD,ERR_MORE_THAN_ONE_RECORD,...
/// ERR_NOT_GOT_RECORD indicate that not got any recode from the database
/// ERR_MORE_THAN_ONE_RECORD indicate that got more than one record from database
func (engine *Engine) SelectOneTpl(dest interface{}, key string, param interface{}) error {
	session := engine.NewSession()
	defer session.Close()
	return session.SelectOneTpl(dest, key, param)
}

/// register a sql template to replace the default, default use go text/template
/// @param tb: the template builder
func (engine *Engine) RegisterTemplate(tb TemplateBuilder, sqlDir string) {
	tplBuilder = tb
	err := engine.initSql(sqlDir)
	if err != nil {
		panic(err)
	}
}

/// init the *.goxml files to map
/// @param sqlDir: the *.goxml file location
func (engine *Engine) initSql(sqlDir string) error {
	files, err := GetFiles(sqlDir)
	if err != nil {
		return err
	}

	for _, f := range files {
		err = engine.initSqlMap(f)
		if err != nil {
			return err
		}
	}

	return nil
}

/// load the *goxml file content to map
/// @param file: the *.goxml file path
func (engine *Engine) initSqlMap(file string) error {
	bts, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	m, err := engine.parse(bts)
	if err != nil {
		return err
	}
	if m != nil && len(m) > 0 {
		for k, v := range m {
			vv := sqlMap[k]
			if vv != nil {
				return errors.New("the *.goxml map key is repeat: " + k)
			} else {
				sqlMap[k] = &SqlTemplate{sql: v}
			}
		}
	}

	return nil
}

/// parse the *.goxml file
func (engine *Engine) parse(xml []byte) (map[string]string, error) {
	ret := map[string]string{}

	engine.ShowSQL()

	doc := etree.NewDocument()
	err := doc.ReadFromBytes(xml)
	if err != nil {
		return ret, err
	}

	sm := doc.SelectElement("sqlmap")
	if sm == nil {
		return nil, errors.New("the sqlmap element is not found")
	}

	namespace := sm.SelectAttrValue("namespace", DefaultNamespace)
	if namespace == "" {
		namespace = DefaultNamespace
	}

	els := sm.SelectElements("sql")
	if els == nil || len(els) < 1 {
		return ret, nil
	}

	for _, e := range els {
		id := e.SelectAttrValue("id", "")
		if id == "" {
			return ret, errors.New(namespace + " has sql not have ID")
		}
		fullId := namespace + "." + id
		if ret[fullId] == fullId {
			return ret, errors.New(namespace + "." + fullId + " repeat")
		}
		val := e.Text()
		val = strings.Replace(val, "\n", " ", -1)
		val = strings.Trim(val, "\n")
		val = strings.TrimSpace(val)
		ret[fullId] = val
	}

	return ret, nil
}

/// get all files from the dirPath
/// @param dirPath: the files dir
func GetFiles(dirPth string) (files []string, err error) {
	var dirs []string
	dir, err := ioutil.ReadDir(dirPth)
	if err != nil {
		return nil, err
	}

	PthSep := string(os.PathSeparator)
	for _, fi := range dir {
		if fi.IsDir() {
			dirs = append(dirs, dirPth+PthSep+fi.Name())
			fs, err := GetFiles(dirPth + PthSep + fi.Name())
			if err != nil {
				return nil, err
			} else {
				files = append(files, fs...)
			}
		} else {
			files = append(files, dirPth+PthSep+fi.Name())
		}
	}

	return files, nil
}

