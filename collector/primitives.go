package collector

import (
	"fmt"
	"os/exec"
	"strings"
)

type Data struct {
	OutputBase string    `json:"output_base"`
	Packages   []Package `json:"packages,omitempty"`
}

type Package struct {
	Name  string `json:"name,omitempty"`
	Files []File `json:"files,omitempty"`
}

type File struct {
	Name             string            `json:"name,omitempty"`
	Package          string            `json:"package,omitempty"`
	Imports          []Import          `json:"imports,omitempty"`
	TypeDefs         []TypeDefinition  `json:"type_defs,omitempty"`
	Structs          []Struct          `json:"structs,omitempty"`
	Interfaces       []Interface       `json:"interfaces,omitempty"`
	Funcs            []Func            `json:"funcs,omitempty"`
	Consts           []Const           `json:"consts,omitempty"`
	Vars             []Var             `json:"vars,omitempty"`
	Comments         []Comment         `json:"comments,omitempty"`
	MagicComments    []MagicComment    `json:"magic_comments,omitempty"`
	GenerateComments []GenerateComment `json:"generate_comments,omitempty"`
	BuildTags        []Constraint      `json:"build_tags,omitempty"`
}

type StructField struct {
	Indirect         bool              `json:"indirect,omitempty"`
	Embed            bool              `json:"embed,omitempty"`
	IsMap            bool              `json:"is_map,omitempty"`
	IsExported       bool              `json:"is_exported,omitempty"`
	IsInterface      bool              `json:"is_interface,omitempty"`
	IsSlice          bool              `json:"is_slice,omitempty"`
	IsArray          bool              `json:"is_array,omitempty"`
	ArrayLen         string            `json:"array_length,omitempty"`
	Name             string            `json:"name,omitempty"`
	Doc              Comment           `json:"doc,omitempty"`
	Comment          Comment           `json:"comment,omitempty"`
	MagicComments    []MagicComment    `json:"magic_comments,omitempty"`
	GenerateComments []GenerateComment `json:"generate_comments,omitempty"`
	Type             interface{}       `json:"field_type,omitempty"`
	Tag              string            `json:"tag,omitempty"`
}

type Method struct {
	IsExported       bool              `json:"is_exported,omitempty"`
	Name             string            `json:"name,omitempty"`
	Doc              Comment           `json:"doc,omitempty"`
	Comment          Comment           `json:"comment,omitempty"`
	MagicComments    []MagicComment    `json:"magic_comments,omitempty"`
	GenerateComments []GenerateComment `json:"generate_comments,omitempty"`
	Receiver         string            `json:"receiver,omitempty"`
	ReceiverIndirect bool              `json:"receiver_indirect,omitempty"`
	Params           []Value           `json:"params,omitempty"`
	Results          []Value           `json:"results,omitempty"`
}

type Struct struct {
	IsExported       bool              `json:"is_exported,omitempty"`
	Name             string            `json:"name,omitempty"`
	Doc              Comment           `json:"doc,omitempty"`
	Comment          Comment           `json:"comment,omitempty"`
	MagicComments    []MagicComment    `json:"magic_comments,omitempty"`
	GenerateComments []GenerateComment `json:"generate_comments,omitempty"`
	Fields           []StructField     `json:"fields,omitempty"`
	Methods          []Method          `json:"methods,omitempty"`
}

type TypeDefinition struct {
	IsExported       bool              `json:"is_exported,omitempty"`
	Name             string            `json:"name,omitempty"`
	Type             string            `json:"type,omitempty"`
	Doc              Comment           `json:"doc,omitempty"`
	Comment          Comment           `json:"comment,omitempty"`
	MagicComments    []MagicComment    `json:"magic_comments,omitempty"`
	GenerateComments []GenerateComment `json:"generate_comments,omitempty"`
	Methods          []Method          `json:"methods,omitempty"`
}

type Map struct {
	KeyType   string   `json:"key_type,omitempty"`
	ValueType MapValue `json:"value_type,omitempty"`
}

func (m Map) String() string {
	return fmt.Sprintf("map[%s]%s", m.KeyType, m.ValueType)
}

type MapValue struct {
	Name  string      `json:"name,omitempty"`
	Value interface{} `json:"value,omitempty"`
}

type Func struct {
	IsExported       bool              `json:"is_exported,omitempty"`
	Name             string            `json:"name,omitempty"`
	Doc              Comment           `json:"doc,omitempty"`
	Comment          Comment           `json:"comment,omitempty"`
	MagicComments    []MagicComment    `json:"magic_comments,omitempty"`
	GenerateComments []GenerateComment `json:"generate_comments,omitempty"`
	Params           []Value           `json:"params,omitempty"`
	Results          []Value           `json:"results,omitempty"`
}

type Channel struct {
	Type     interface{} `json:"type,omitempty"`
	RecvOnly bool        `json:"recv_only,omitempty"`
	SendOnly bool        `json:"send_only,omitempty"`
}

type ValueType struct {
	Kind  string      `json:"kind,omitempty"`
	Value interface{} `json:"value,omitempty"`
}

type Value struct {
	Name *string `json:"name,omitempty"`
	Type string  `json:"type,omitempty"`
}

func (v Value) String() string {
	if v.Name == nil {
		n := "<nil>"
		v.Name = &n
	}
	return fmt.Sprintf("Value{%s, %s}", *v.Name, v.Type)
}

type Interface struct {
	IsExported       bool              `json:"is_exported,omitempty"`
	Embed            bool              `json:"embed,omitempty"`
	Name             string            `json:"name,omitempty"`
	Doc              Comment           `json:"doc,omitempty"`
	Comment          Comment           `json:"comment,omitempty"`
	MethodSet        []InterfaceField  `json:"method_set,omitempty"`
	MagicComments    []MagicComment    `json:"magic_comments,omitempty"`
	GenerateComments []GenerateComment `json:"generate_comments,omitempty"`
}

type InterfaceField interface{}

type Import struct {
	Name             string            `json:"name,omitempty"`
	Path             string            `json:"path,omitempty"`
	Doc              Comment           `json:"doc,omitempty"`
	Comment          Comment           `json:"comment,omitempty"`
	MagicComments    []MagicComment    `json:"magic_comments,omitempty"`
	GenerateComments []GenerateComment `json:"generate_comments,omitempty"`
}

type MagicComment struct {
	Pragma string `json:"pragma,omitempty"` // noinline
	Raw    string `json:"raw,omitempty"`    // go:noinline
}

type GenerateComment struct {
	Command string `json:"command,omitempty"` // goyacc -o gopher.go -p parser gopher.y
	Raw     string `json:"raw,omitempty"`     // go:generate goyacc -o gopher.go -p parser gopher.y
}

func (g GenerateComment) Cmd() (*exec.Cmd, error) {
	cmd := strings.Split(g.Command, " ")
	if len(cmd) == 0 {
		return nil, fmt.Errorf("cmd error, not enough args: comment = %s", g.Raw)
	}

	return exec.Command(cmd[0], cmd[1:]...), nil
}

type Comment struct {
	Content string `json:"content,omitempty"`
}

// Lines converts any comment into a slice of strings based on their logical
// line-based grouping.
func (c Comment) Lines() []string {
	const pref = "/*"
	const suff = "*/"
	const basic = "// "

	com := c.Content
	switch {
	// check if multi-line /* ... */ style comment
	case strings.HasPrefix(com, pref) && strings.HasSuffix(com, suff):
		com = strings.Trim(com, pref+suff)
		com = strings.Replace(com, "\n\n", "\n", -1)
		com = strings.TrimPrefix(com, "\n")
		com = strings.TrimSuffix(com, "\n")
		return strings.Split(com, "\n")

	// check if basic comment, prefixed with //(+space)
	case strings.HasPrefix(com, basic):
		return strings.Split(com, basic)
	}

	return nil
}

type Const struct {
	IsExported       bool              `json:"is_exported,omitempty"`
	Name             string            `json:"name,omitempty"`
	Type             string            `json:"type,omitempty"`
	Value            interface{}       `json:"value,omitempty"`
	Doc              Comment           `json:"doc,omitempty"`
	Comment          Comment           `json:"comment,omitempty"`
	MagicComments    []MagicComment    `json:"magic_comments,omitempty"`
	GenerateComments []GenerateComment `json:"generate_comments,omitempty"`
}

type Var struct {
	IsExported       bool              `json:"is_exported,omitempty"`
	Name             string            `json:"name,omitempty"`
	Type             string            `json:"type,omitempty"`
	Value            interface{}       `json:"value,omitempty"`
	Doc              Comment           `json:"doc,omitempty"`
	Comment          Comment           `json:"comment,omitempty"`
	MagicComments    []MagicComment    `json:"magic_comments,omitempty"`
	GenerateComments []GenerateComment `json:"generate_comments,omitempty"`
}

// Constraint holds the options of a build tag.
// +build linux,386 darwin,!cgo
//        |-------| |---------|
//         option      option
type Constraint struct {
	Options []string `json:"options,omitempty"`
}

func (c Constraint) String() string {
	return fmt.Sprintf("%s %s", buildPrefix, strings.Join(c.Options, " "))
}
