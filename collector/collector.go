package collector

import (
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"strings"
)

const (
	slashes      = `//`
	magicPrefix  = slashes + `go:`
	gogenPrefix  = magicPrefix + `generate`
	buildPrefix  = slashes + ` +build`
	ellipsis     = "..."
	interfaceLit = "interface{}"
	literalLit   = "literal"
	typeLit      = "type"
	mapLit       = "map"
	arrayLit     = "array"
	sliceLit     = "slice"
	funcLit      = "func"
	chanLit      = "chan"
)

type FileCollector struct {
	Imports          []Import
	Consts           []Const
	Vars             []Var
	Structs          []Struct
	TypeDefs         []TypeDefinition
	Interfaces       []Interface
	Funcs            []Func
	Comments         []Comment
	MagicComments    []MagicComment
	GenerateComments []GenerateComment
	BuildTags        []Constraint
}

func (c *FileCollector) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return c
	}

	// only collect declarations at file-level
	file, ok := node.(*ast.File)
	if !ok {
		return c
	}

	// collect all file-level comments, including doc comments associated with
	// other file-level declarations
	// additionally, check each comment for special comment prefixes and save
	// matches accordingly, e.g. // +build, //go:
	c.collectComments(file)

	// collect all file-level imports
	c.collectImports(file)

	unresolvedTypes := make(map[string]*TypeDefinition)
	structs := make(map[string]*Struct)
	types := make(map[string]*TypeDefinition)
	interfaces := make([]Interface, 0)
	funcs := make([]Func, 0)
	vars := make([]Var, 0)
	consts := make([]Const, 0)

	// iterate through declarations within the file
	for _, decl := range file.Decls {
		switch n := decl.(type) {
		case *ast.FuncDecl:
			magicComments, generateComments := specialComments(n.Doc)

			// find methods on receiver types (will have a Recv prop)
			method := Method{
				Name:             n.Name.Name,
				IsExported:       isExported(n.Name),
				Doc:              normalizeComment(n.Doc),
				MagicComments:    magicComments,
				GenerateComments: generateComments,
			}
			if n.Recv != nil {
				var exportedRecv bool
				// get the receiver's name and check if it is a pointer
				for i := range n.Recv.List {
					if recv, ok := n.Recv.List[i].Type.(*ast.Ident); ok {
						exportedRecv = isExported(recv)
						method.Receiver = recv.Name
						break
					}
				}
				for i := range n.Recv.List {
					if ptr, ok := n.Recv.List[i].Type.(*ast.StarExpr); ok {
						method.Receiver = ptr.X.(*ast.Ident).Name
						method.ReceiverIndirect = true
						break
					}
				}

				// if the receiver type has already been encountered
				// and stored in our unresolved type map, add this method to it
				if t, ok := unresolvedTypes[method.Receiver]; ok {
					t.IsExported = exportedRecv
					t.Methods = append(t.Methods, method)
				} else {
					// otherwise, create the type def and insert it into
					// the struct map
					unresolvedTypes[method.Receiver] = &TypeDefinition{
						Name:       method.Receiver,
						IsExported: exportedRecv,
						Methods:    []Method{method},
					}
				}
				continue
			}

			// if the func has no reciever, collect it as a basic function
			funcs = append(funcs, Func{
				Name:             n.Name.Name,
				Doc:              normalizeComment(n.Doc),
				IsExported:       isExported(n.Name),
				Params:           funcFields(n.Type.Params),
				Results:          funcFields(n.Type.Results),
				MagicComments:    magicComments,
				GenerateComments: generateComments,
			})

		case *ast.GenDecl:
			for _, spec := range n.Specs {
				switch s := spec.(type) {
				case *ast.ValueSpec:
					// find and stash values including file-level constants and
					// variables
					for _, ident := range s.Names {
						if ident.Obj != nil {
							magic, generate := specialComments(s.Doc)
							var valType string
							for _, val := range s.Values {
								if lit, ok := val.(*ast.BasicLit); ok {
									valType = lit.Kind.String()
								}
							}
							switch ident.Obj.Kind {
							case ast.Var:
								val := value(s)
								if val == nil {
									continue
								}
								vars = append(vars, Var{
									IsExported:       isExported(ident),
									Name:             ident.Name,
									Value:            val,
									Type:             valType,
									Doc:              normalizeComment(s.Doc),
									Comment:          normalizeComment(s.Comment),
									MagicComments:    magic,
									GenerateComments: generate,
								})

							case ast.Con:
								val := value(s)
								if val == nil {
									continue
								}
								consts = append(consts, Const{
									IsExported:       isExported(ident),
									Name:             ident.Name,
									Value:            val,
									Type:             valType,
									Doc:              normalizeComment(s.Doc),
									Comment:          normalizeComment(s.Comment),
									MagicComments:    magic,
									GenerateComments: generate,
								})
							}
						}
					}

				case *ast.TypeSpec:
					// find and stash the structs
					if strct, ok := s.Type.(*ast.StructType); ok {
						var fields []StructField
						for _, field := range strct.Fields.List {
							var (
								fType         interface{}
								indirect      bool
								isArray       bool
								arrayLen      string
								isSlice       bool
								isMap         bool
								exportedField bool
							)

							fName := identName(field.Names)
							for _, nm := range field.Names {
								if isExported(nm) {
									exportedField = true
									break
								}
							}

							switch t := field.Type.(type) {
							case *ast.SelectorExpr:
								typ := t.Sel.Name
								slct := t.X.(*ast.Ident).Name
								fType = ValueType{
									Kind:  typeLit,
									Value: slct + "." + typ,
								}
							case *ast.StarExpr:
								indirect = true
								// check if we have a selector (within package, etc)
								// expression prepending the identifier
								if sel, ok := t.X.(*ast.SelectorExpr); ok {
									typ := sel.Sel.Name
									slct := sel.X.(*ast.Ident).Name
									fType = ValueType{
										Kind:  typeLit,
										Value: slct + "." + typ,
									}
								} else {
									switch typ := t.X.(type) {
									case *ast.Ident:
										fType = ValueType{
											Kind:  typeLit,
											Value: typ.Name,
										}

									case *ast.ArrayType:
										var kind string
										if typ.Len == nil {
											kind = sliceLit
										} else {
											kind = arrayLit
										}

										fType = ValueType{
											Kind:  kind,
											Value: typeName(typ.Elt),
										}
									}
								}

							case *ast.Ident:
								if star, ok := field.Type.(*ast.StarExpr); ok {
									indirect = true
									fType = ValueType{
										Kind:  typeLit,
										Value: star.X.(*ast.Ident).Name,
									}
								} else {
									fType = field.Type.(*ast.Ident).Name
								}

							case *ast.ChanType:
								fType = ValueType{
									Kind:  chanLit,
									Value: channelType(t),
								}

							case *ast.MapType:
								isMap = true
								fType = ValueType{
									Kind:  mapLit,
									Value: mapType(t),
								}

							case *ast.ArrayType:
								if t.Len == nil {
									isSlice = true
								} else {
									isArray = true
									switch l := t.Len.(type) {
									case *ast.BasicLit:
										arrayLen = l.Value
									case *ast.Ident:
										arrayLen = l.Name
									}
								}
								switch fieldType := t.Elt.(type) {
								case *ast.StarExpr:
									indirect = true
									fType = ValueType{
										Kind:  typeLit,
										Value: typeName(fieldType.X),
									}

								case *ast.InterfaceType:
									fType = interfaceLit

								case *ast.Ident:
									fType = fieldType.Name

								case *ast.MapType:
									fType = ValueType{
										Kind:  mapLit,
										Value: mapType(fieldType),
									}

								case *ast.ChanType:
									fType = ValueType{
										Kind:  chanLit,
										Value: channelType(fieldType),
									}
								}
							}

							magic, generate := specialComments(field.Doc)
							fields = append(fields, StructField{
								Name:             fName,
								Type:             fType,
								Tag:              fieldTag(field),
								Embed:            fName == "",
								Indirect:         indirect,
								IsArray:          isArray,
								ArrayLen:         arrayLen,
								IsSlice:          isSlice,
								IsMap:            isMap,
								IsExported:       exportedField,
								Doc:              normalizeComment(field.Doc),
								Comment:          normalizeComment(field.Comment),
								MagicComments:    magic,
								GenerateComments: generate,
							})
						}

						doc := normalizeComment(n.Doc)
						comment := normalizeComment(s.Comment)

						if strct, ok := structs[s.Name.Name]; ok {
							strct.Doc = doc
							strct.Comment = comment
							strct.Fields = fields
						} else {
							structs[s.Name.Name] = &Struct{
								Name:    s.Name.Name,
								Doc:     doc,
								Comment: comment,
								Fields:  fields,
							}
						}
					}

					// find and stash the interfaces
					if iface, ok := s.Type.(*ast.InterfaceType); ok {
						magic, generate := specialComments(s.Doc)
						interfaces = append(interfaces, Interface{
							IsExported:       isExported(s.Name),
							Name:             s.Name.Name,
							Doc:              normalizeComment(s.Doc),
							Comment:          normalizeComment(s.Comment),
							MethodSet:        methodSet(iface),
							MagicComments:    magic,
							GenerateComments: generate,
						})
					}

					// find and stash other type definitions
					if ident, ok := s.Type.(*ast.Ident); ok {
						magic, generate := specialComments(s.Doc)

						def := &TypeDefinition{
							IsExported:       isExported(s.Name),
							Name:             s.Name.Name,
							Type:             ident.Name,
							Doc:              normalizeComment(s.Doc),
							Comment:          normalizeComment(s.Comment),
							MagicComments:    magic,
							GenerateComments: generate,
						}

						// if the type def was already encountered from finding
						// one of its methods, add the detailed data to the map
						// to the existing type def
						if td, ok := types[s.Name.Name]; ok {
							def.Methods = td.Methods
							types[s.Name.Name] = def
						} else {
							types[s.Name.Name] = def
						}
					}
				}
			}
		}
	}

	for k, v := range structs {
		// capture methods from unresolved type def map and provide to the
		// actual struct encountered
		if utd, ok := unresolvedTypes[k]; ok {
			v.Methods = append(v.Methods, utd.Methods...)
		}

		c.Structs = append(c.Structs, *v)
	}

	for k, v := range types {
		// capture methods from unresolved type def map and provide to the
		// actual type definition encountered
		if utd, ok := unresolvedTypes[k]; ok {
			v.Methods = append(v.Methods, utd.Methods...)
		}

		c.TypeDefs = append(c.TypeDefs, *v)
	}

	c.Vars = vars
	c.Consts = consts
	c.Funcs = funcs
	c.Interfaces = interfaces

	return c
}

func (c *FileCollector) collectImports(file *ast.File) {
	if file == nil {
		return
	}

	for _, imp := range file.Imports {
		var name string
		if imp.Name != nil {
			name = imp.Name.Name
		}
		magic, generate := specialComments(imp.Doc)
		c.Imports = append(c.Imports, Import{
			Name:             name,
			Path:             imp.Path.Value,
			Doc:              normalizeComment(imp.Doc),
			Comment:          normalizeComment(imp.Comment),
			MagicComments:    magic,
			GenerateComments: generate,
		})
	}
}

func (c *FileCollector) collectComments(file *ast.File) {
	if file == nil {
		return
	}

	for _, group := range file.Comments {
		for _, com := range group.List {
			switch {
			case strings.HasPrefix(com.Text, buildPrefix):
				opts := strings.TrimSpace(
					strings.TrimPrefix(com.Text, buildPrefix),
				)
				constraint := Constraint{
					Options: strings.Split(opts, " "),
				}
				c.BuildTags = append(c.BuildTags, constraint)

			case strings.HasPrefix(com.Text, magicPrefix):
				switch nsc := nonStandardComment(com).(type) {
				case *MagicComment:
					if nsc != nil {
						c.MagicComments = append(c.MagicComments, *nsc)
					}

				case *GenerateComment:
					if nsc != nil {
						c.GenerateComments = append(c.GenerateComments, *nsc)
					}
				}
			}
		}
		c.Comments = append(c.Comments, normalizeComment(group))
	}
}

func mapType(m *ast.MapType) Map {
	var kv Map
	switch k := m.Key.(type) {
	case *ast.Ident:
		kv.KeyType = k.Name

	case *ast.BasicLit:
		kv.KeyType = k.Value

	case *ast.StarExpr:
		if selExp, ok := k.X.(*ast.SelectorExpr); ok {
			pkg := selExp.X.(*ast.Ident).Name
			sel := selExp.Sel.Name
			kv.KeyType = fmt.Sprintf("*%s.%s", pkg, sel)
		} else {
			kv.KeyType = "*" + k.X.(*ast.Ident).Name
		}

	case *ast.SelectorExpr:
		pkg := k.X.(*ast.Ident).Name
		sel := k.Sel.Name
		kv.KeyType = fmt.Sprintf("%s.%s", pkg, sel)

	case *ast.InterfaceType:
		kv.KeyType = interfaceLit
	}

	switch v := m.Value.(type) {
	case *ast.Ident:
		kv.ValueType = MapValue{
			Name:  literalLit,
			Value: v.Name,
		}

	case *ast.BasicLit:
		kv.ValueType = MapValue{
			Name:  literalLit,
			Value: v.Value,
		}

	case *ast.StarExpr:
		if selExp, ok := v.X.(*ast.SelectorExpr); ok {
			pkg := selExp.X.(*ast.Ident).Name
			sel := selExp.Sel.Name
			kv.ValueType = MapValue{
				Name:  typeLit,
				Value: fmt.Sprintf("*%s.%s", pkg, sel),
			}
		} else {
			kv.ValueType = MapValue{
				Name:  typeLit,
				Value: "*" + v.X.(*ast.Ident).Name,
			}
		}

	case *ast.SelectorExpr:
		pkg := v.X.(*ast.Ident).Name
		sel := v.Sel.Name
		kv.ValueType = MapValue{
			Name:  typeLit,
			Value: fmt.Sprintf("%s.%s", pkg, sel),
		}

	case *ast.ArrayType:
		var size string
		valTypeName := arrayLit
		if v.Len != nil {
			valTypeName = sliceLit
			switch l := v.Len.(type) {
			case *ast.BasicLit:
				size = l.Value

			case *ast.Ellipsis:
				size = ellipsis
			}

		}
		arrType := typeName(v.Elt)
		kv.ValueType = MapValue{
			Name:  valTypeName,
			Value: fmt.Sprintf("[%s]%s", size, arrType),
		}

	case *ast.MapType:
		kv.ValueType = MapValue{
			Name:  mapLit,
			Value: mapType(v),
		}

	case *ast.FuncType:
		kv.ValueType = MapValue{
			Name: funcLit,
			Value: Func{
				IsExported: false, // documenting that func literal cannot be exported e.g. `func() {}`
				Params:     funcFields(v.Params),
				Results:    funcFields(v.Results),
			},
		}
	case *ast.InterfaceType:
		kv.ValueType = MapValue{
			Name:  interfaceLit,
			Value: interfaceLit,
		}

	case *ast.ChanType:
		kv.ValueType = MapValue{
			Name: chanLit,
			Value: Channel{
				Type:     typeName(v.Value),
				RecvOnly: v.Dir == ast.RECV,
				SendOnly: v.Dir == ast.SEND,
			},
		}
	}

	return kv
}

func channelType(ch *ast.ChanType) Channel {
	return Channel{
		Type:     typeName(ch.Value),
		RecvOnly: ch.Dir == ast.RECV,
		SendOnly: ch.Dir == ast.SEND,
	}
}

func value(s *ast.ValueSpec) interface{} {
	for _, val := range s.Values {
		switch expr := val.(type) {
		case *ast.BasicLit:
			return expr.Value
		case *ast.CompositeLit:
			switch expr := expr.Type.(type) {
			case *ast.BasicLit:
				return expr.Value
			}
		case *ast.StarExpr:
			switch expr := expr.X.(type) {
			case *ast.BasicLit:
				return "*" + expr.Value
			case *ast.CompositeLit:
				switch expr := expr.Type.(type) {
				case *ast.BasicLit:
					return "*" + expr.Value
				}
			}
		}
	}
	return nil
}

func rawExpression(expr ast.Expr) (string, error) {
	buf := &strings.Builder{}
	err := printer.Fprint(buf, token.NewFileSet(), expr)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func specialComments(doc *ast.CommentGroup) ([]MagicComment, []GenerateComment) {
	if doc == nil {
		return nil, nil
	}

	var magicComments []MagicComment
	var generateComments []GenerateComment
	for _, doc := range doc.List {
		switch nsc := nonStandardComment(doc).(type) {
		case *MagicComment:
			if nsc != nil {
				magicComments = append(magicComments, *nsc)
			}
		case *GenerateComment:
			if nsc != nil {
				generateComments = append(generateComments, *nsc)
			}
		}
	}

	return magicComments, generateComments
}

func nonStandardComment(com *ast.Comment) interface{} {
	if com == nil {
		return nil
	}

	switch {
	case strings.HasPrefix(com.Text, gogenPrefix):
		return &GenerateComment{
			Command: strings.TrimSpace(
				strings.TrimPrefix(com.Text, gogenPrefix),
			),
			Raw: com.Text,
		}
	case strings.HasPrefix(com.Text, magicPrefix):
		return &MagicComment{
			Pragma: strings.TrimSpace(
				strings.TrimPrefix(com.Text, magicPrefix),
			),
			Raw: com.Text,
		}
	}
	return nil
}

func methodSet(iface *ast.InterfaceType) []InterfaceField {
	if iface == nil {
		return nil
	}

	var fields []InterfaceField
	for _, field := range iface.Methods.List {
		switch ifaceField := field.Type.(type) {
		case *ast.SelectorExpr:
			pkg := ifaceField.X.(*ast.Ident).Name
			sel := ifaceField.Sel.Name
			embd := Interface{
				Name:       fmt.Sprintf("%s.%s", pkg, sel),
				IsExported: isExported(ifaceField.Sel),
				Embed:      true,
			}
			fields = append(fields, embd)

		case *ast.Ident:
			embd := Interface{
				Name:       ifaceField.Name,
				IsExported: isExported(ifaceField),
				Embed:      true,
			}
			fields = append(fields, embd)

		case *ast.FuncType:
			var name string
			var exported bool
			if field.Names != nil {
				name = typeName(field.Names[0]).(string)
				exported = isExported(field.Names[0])
			}
			fn := Func{
				Name:       name,
				IsExported: exported,
				Doc:        normalizeComment(field.Doc),
				Comment:    normalizeComment(field.Comment),
				Params:     funcFields(ifaceField.Params),
				Results:    funcFields(ifaceField.Results),
			}
			fields = append(fields, fn)
		}
	}

	return fields
}

func funcFields(list *ast.FieldList) []Value {
	if list == nil {
		return nil
	}
	if list.List == nil {
		return nil
	}

	var vals []Value
	for _, part := range list.List {
		var name *string
		if part.Names != nil {
			name = &part.Names[0].Name
		}
		vals = append(vals, Value{
			Name: name,
			Type: typeName(part.Type).(string),
		})
	}

	return vals
}

func typeName(expr ast.Expr) interface{} {
	str := &strings.Builder{}
	printer.Fprint(str, token.NewFileSet(), expr)
	return str.String()
}

func arrayTypeName(arr *ast.ArrayType) string {
	const tmpl = `[%s]%s`

	arrType := typeName(arr.Elt)
	if arr.Len == nil {
		return fmt.Sprintf(tmpl, "", arrType)
	}

	size := arr.Len.(*ast.BasicLit).Value
	return fmt.Sprintf(tmpl, size, arrType)
}

func normalizeComment(docs *ast.CommentGroup) Comment {
	if docs == nil {
		return Comment{Content: ""}
	}

	var all []string
	for _, c := range docs.List {
		// ignore non-standard comments, which will be available as properties
		// objects where appropriate
		switch nonStandardComment(c).(type) {
		case *MagicComment, *GenerateComment:
			continue
		}

		all = append(all, strings.TrimSpace(c.Text))
	}

	return Comment{Content: strings.Join(all, "")}
}

func identName(names []*ast.Ident) string {
	if names == nil {
		return ""
	}

	for _, name := range names {
		return name.Name
	}

	return ""
}

func fieldTag(field *ast.Field) string {
	if field.Tag != nil {
		return field.Tag.Value
	}

	return ""
}

func isExported(ident *ast.Ident) bool {
	if ident.Name == "" || ident == nil {
		return false
	}

	return ast.IsExported(ident.Name)
}
