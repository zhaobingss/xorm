package xorm

import (
	"io"
	"text/template"
)

/// template builder interface
type TemplateBuilder interface {
	New(name, content string) (Template, error)
}

/// template execute interface
type TemplateExecutor interface {
	Execute(wr io.Writer, param interface{}) error
}

/// template interface
type Template interface {
	TemplateBuilder
	TemplateExecutor
}

/// the default Template impl
type DefaultTemplate struct {
	tpl *template.Template
}

/// create a new template
/// name: the template name
/// content: the template content that the template execute will be use
func (dt *DefaultTemplate) New(name, content string) (Template, error) {
	tpl := template.New(name)
	tpl, err := tpl.Parse(content)
	if err != nil {
		return nil, err
	}
	dtp := &DefaultTemplate{
		tpl:tpl,
	}
	return dtp, nil
}

/// execute template and write the result to a writer
/// wr: the result will be write
/// param: the template execute param
func (dt *DefaultTemplate) Execute(wr io.Writer, param interface{}) error {
	return dt.tpl.Execute(wr, param)
}
