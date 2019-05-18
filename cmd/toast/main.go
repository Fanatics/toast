package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"

	"github.com/Fanatics/toast/collector"
	"github.com/tidwall/sjson"
)

const toastPrefix = "[toast]"

var plugins *plugin

func main() {
	input := flag.String("input", ".", "input directory from where to parse Go code")
	debug := flag.Bool("debug", false, "write data from parsed AST to stdout, skips plugins")
	flag.Var(plugins, "plugin", "executable plugin for toast to invoke, and the output base directory for files to be written")
	flag.Parse()

	fset := token.NewFileSet()
	data := &collector.Data{}

	err := filepath.Walk(*input, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			log.Fatal("recursive walk error:", err)
		}
		// skip over files, only continue into directories for parser to enter
		if !fi.IsDir() {
			return nil
		}

		pkgs, err := parser.ParseDir(fset, path, nil, parser.ParseComments)
		if err != nil {
			log.Fatalf("parse dir error: %v\n", err)
		}
		for _, pkg := range pkgs {
			p := collector.Package{
				Name: pkg.Name,
			}
			for _, file := range pkg.Files {
				c := &collector.FileCollector{}
				ast.Walk(c, file)
				f := collector.File{
					Name:             fset.Position(file.Pos()).Filename,
					Package:          pkg.Name,
					Imports:          c.Imports,
					BuildTags:        c.BuildTags,
					Comments:         c.Comments,
					MagicComments:    c.MagicComments,
					GenerateComments: c.GenerateComments,
					Consts:           c.Consts,
					Vars:             c.Vars,
					Structs:          c.Structs,
					TypeDefs:         c.TypeDefs,
					Interfaces:       c.Interfaces,
					Funcs:            c.Funcs,
				}
				p.Files = append(p.Files, f)
			}
			data.Packages = append(data.Packages, p)
		}

		return nil
	})
	if err != nil {
		exitWithMessage("filepath walk error", err)
	}

	// debug mode enables users to inspect the raw JSON on the command line
	if *debug {
		b, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			exitWithMessage("(debug) JSON encode error", err)
		}
		// write the data to stdout and skip the plugin execution
		fmt.Println(string(b))
		return
	}

	b, err := json.Marshal(data)
	if err != nil {
		exitWithMessage("JSON encode error", err)
		os.Exit(1)
	}

	err = plugins.each(func(i int, p *plugin) error {
		// replace the output base value for each plugin rather than decoding,
		// re-assigning the value, and re-encoding
		b, err := sjson.SetBytes(b, "output_base", []byte(p.outputDir))
		if err != nil {
			return err
		}
		// set the plugin into a runner and execute it, passing in the data
		exe := &runner{
			p:    p,
			data: bytes.NewReader(b),
		}
		if err := exe.run(); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		// err is a collection of errors, one per line, from all of the plugins
		fmt.Println(toastPrefix, "accumulated plugin errors:")
		fmt.Println(err)
		os.Exit(1)
	}
}

func exitWithMessage(msg string, err error) {
	fmt.Println(toastPrefix, msg, err)
	os.Exit(1)
}
