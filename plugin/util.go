package plugin

import (
	"bytes"
	"go/format"
	htmltmpl "html/template"
	"io"
	"path"
	"text/template"
)

// Gofmt formats Go source code.
func Gofmt(src []byte) ([]byte, error) {
	return format.Source(src)
}

// GofmtReadWriter formats an io.ReadWriter, such as an *os.File.
func GofmtReadWriter(rw io.ReadWriter) (io.ReadWriter, error) {
	buf := bytes.Buffer{}
	_, err := io.Copy(&buf, rw)
	if err != nil {
		return nil, err
	}

	fmtd, err := Gofmt(buf.Bytes())
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(fmtd), nil
}

// OutputTemplate executes a text template using the provided data and writes it
// to the destination io.Writer.
func (p *plugin) OutputTemplate(dst io.Writer, templatePath string, data interface{}) error {
	return template.Must(
		template.New(path.Base(templatePath)).ParseFiles(templatePath),
	).Execute(dst, data)
}

// OutputTemplateHTML executes an HTML template using the provided data and
// writes it to the destination io.Writer.
func (p *plugin) OutputTemplateHTML(dst io.Writer, templatePath string, data interface{}) error {
	return htmltmpl.Must(
		htmltmpl.New(path.Base(templatePath)).ParseFiles(templatePath),
	).Execute(dst, data)
}
